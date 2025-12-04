package usbwallet

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// Add support for ledger sign personal message.
	ledgerP1InitPersonalMessageData ledgerParam1 = 0x00 // First chunk of Personal Message data
	ledgerP1ContPersonalMessageData ledgerParam1 = 0x80 // Next chunk of Personal Message data
	ledgerOpSignPersonalMessage     ledgerOpcode = 0x08 // Signs an ethereum personal message
)

// signMessage implements accounts.Wallet
// Used for EIP191 (personal signing).
func (w *wallet) signMessage(account accounts.Account, message []byte) ([]byte, error) {
	w.stateLock.RLock() // Comms have own mutex, this is for the state fields
	defer w.stateLock.RUnlock()

	// If the wallet is closed, abort
	if w.device == nil {
		return nil, accounts.ErrWalletClosed
	}
	// Make sure the requested account is contained within
	path, ok := w.paths[account.Address]
	if !ok {
		return nil, accounts.ErrUnknownAccount
	}
	// All infos gathered and metadata checks out, request signing
	<-w.commsLock
	defer func() { w.commsLock <- struct{}{} }()

	// Ensure the device isn't screwed with while user confirmation is pending
	// TODO(karalabe): remove if hotplug lands on Windows
	w.hub.commsLock.Lock()
	w.hub.commsPend++
	w.hub.commsLock.Unlock()

	defer func() {
		w.hub.commsLock.Lock()
		w.hub.commsPend--
		w.hub.commsLock.Unlock()
	}()
	signature, err := w.driver.SignPersonalMessage(path, message)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (w *ledgerDriver) SignPersonalMessage(path accounts.DerivationPath, message []byte) ([]byte, error) {
	// If the Ethereum app doesn't run, abort
	if w.offline() {
		return nil, accounts.ErrWalletClosed
	}
	// Ensure the wallet is capable of signing the given transaction
	if w.version[0] < 1 && w.version[1] < 2 {
		//lint:ignore ST1005 brand name displayed on the console
		return nil, fmt.Errorf("version error: Ledger version >= 1.2.0 required for personal signing (found version v%d.%d.%d)", w.version[0], w.version[1], w.version[2])
	}
	return w.ledgerSignPersonalMessage(path, message)
}

func (w *ledgerDriver) ledgerSignPersonalMessage(derivationPath []uint32, message []byte) ([]byte, error) {
	// Flatten the derivation path into the Ledger request
	path := make([]byte, 1+4*len(derivationPath))
	path[0] = byte(len(derivationPath))
	for i, component := range derivationPath {
		binary.BigEndian.PutUint32(path[1+4*i:], component)
	}
	var messageLength [4]byte
	// G115 check
	msgLen := len(message)
	if msgLen > math.MaxUint32 {
		return nil, fmt.Errorf("message length %d exceeds uint32 max", msgLen)
	}
	binary.BigEndian.PutUint32(messageLength[:], uint32(msgLen)) //nolint:gosec // G115: overflow checked above
	payload := append(path, messageLength[:]...)
	payload = append(payload, message...)
	// Send the request and wait for the response
	var (
		op    = ledgerP1InitPersonalMessageData
		reply []byte
		err   error
	)
	fmt.Println("Derivation path: " + string(path))
	// Chunk size selection to mitigate an underlying RLP deserialization issue on the ledger app.
	// https://github.com/LedgerHQ/app-ethereum/issues/409
	chunk := 255
	//nolint:revive // alow empty block
	for ; len(payload)%chunk <= ledgerEip155Size; chunk-- {
	}

	for len(payload) > 0 {
		// Calculate the size of the next data chunk
		if chunk > len(payload) {
			chunk = len(payload)
		}
		// Send the chunk over, ensuring it's processed correctly
		reply, err = w.ledgerExchange(ledgerOpSignPersonalMessage, op, 0, payload[:chunk])
		if err != nil {
			return nil, err
		}
		// Shift the payload and ensure subsequent chunks are marked as such
		payload = payload[chunk:]
		op = ledgerP1ContPersonalMessageData
	}

	// Extract the Ethereum signature and do a sanity validation
	if len(reply) != crypto.SignatureLength {
		return nil, fmt.Errorf("reply lacks signature: reploy %v", reply)
	}
	signature := append(reply[1:], reply[0])
	return signature, nil
}
