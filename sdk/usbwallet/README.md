# Notes

- Lifted from Geth v1.14.7 source code accounts/usbwallet directory.
  - Note that Geth < 1.14 version suffers from this issue described and patched here https://github.com/ethereum/go-ethereum/pull/28945.
- Removed trezor
- Modified to add EIP 191 support (SignPersonalMessage). The Geth library does not implement EIP 191
intentionally, as it is less secure than its successor EIP 712. However, in the case of MCMS we explicitly 
want the cross chain replayability possibility of EIP 191. Luckily the ledger communication  
protocol is already fully supported. Aside from adding the new methods to the hub and wallet interfaces,
the diff is localized to eip191.go.