package mcms

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/mcms/pkg/errors"
	"github.com/smartcontractkit/mcms/pkg/internal/manifest"
	"github.com/smartcontractkit/mcms/pkg/proposal/mcms/types"
)

// jsonDecoder reads and decodes manifest JSON data from an input stream into an MCMS proposal.
type jsonDecoder struct {
	r io.Reader
}

// newJSONDecoder returns a new decode that reads from r.
func newJSONDecoder(r io.Reader) *jsonDecoder {
	return &jsonDecoder{r: r}
}

// Decode decodes the JSON data into the MCMS proposal.
func (dec *jsonDecoder) Decode(prop *MCMSProposal) error {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(dec.r); err != nil {
		return err
	}

	data := buf.Bytes()
	ver, err := manifest.GetVersion(data)
	if err != nil {
		return err
	}

	switch ver {
	case "v1":
		err = dec.decodeV1(data, prop)
	default:
		return &errors.InvalidVersionError{ReceivedVersion: ver}
	}

	return err
}

func (dec *jsonDecoder) decodeV1(data []byte, prop *MCMSProposal) error {
	// Unmarshal the JSON data into the V1 manifest struct
	var m manifest.ProposalV1
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Assign the fields from the V1 manifest to the MCMS proposal
	*prop = MCMSProposal{
		Version:              m.Version,
		Kind:                 m.Kind,
		ValidUntil:           m.ValidUntil,
		OverridePreviousRoot: m.OverridePreviousRoot,
		Description:          m.Description,
		Signatures:           make([]Signature, 0, len(m.Signatures)),
		ChainMetadata:        make(map[types.ChainIdentifier]ChainMetadata),
		Transactions:         make([]types.ChainOperation, 0, len(m.Transactions)),
	}

	// Map the signatures
	for _, sig := range m.Signatures {
		prop.Signatures = append(prop.Signatures, Signature{
			R: sig.R,
			S: sig.S,
			V: sig.V,
		})
	}

	// Map the chain identifiers to the chain metadata
	for sel, md := range m.ChainMetadata {
		prop.ChainMetadata[types.ChainIdentifier(sel)] = ChainMetadata{
			StartingOpCount: md.StartingOpCount,
			MCMAddress:      common.HexToAddress(md.MCMAddress),
		}
	}

	// Map the transactions to the chain operations
	for _, t := range m.Transactions {
		chainOp := types.ChainOperation{
			ChainIdentifier: types.ChainIdentifier(t.ChainSelector),
			Operation: types.Operation{
				To:               common.HexToAddress(t.To),
				Data:             t.Data,
				AdditionalFields: t.AdditionalFields, // TODO: Can we convert this to an interface or any here so we don't have to expose json to the sdks?
				ContractType:     t.ContractType,
				Tags:             t.Tags,
			},
		}

		prop.Transactions = append(prop.Transactions, chainOp)
	}

	return nil
}

// jsonEncoder writes an MCMS proposal to an output stream in JSON format.
type jsonEncoder struct {
	w io.Writer
}

// newJSONEncoder returns a new encoder that writes to w.
func newJSONEncoder(w io.Writer) *jsonEncoder {
	return &jsonEncoder{w: w}
}

// Encode encodes the MCMS proposal into JSON format.
func (enc *jsonEncoder) Encode(prop *MCMSProposal) error {
	var (
		bytes []byte
		err   error
	)
	switch prop.Version {
	case "v1":
		bytes, err = enc.encodeV1(prop)
	default:
		return &errors.InvalidVersionError{ReceivedVersion: prop.Version}
	}

	if err != nil {
		return err
	}

	_, err = enc.w.Write(bytes)

	return err

}

// encodeV1 encodes the MCMS proposal into a V1 manifest.
func (enc *jsonEncoder) encodeV1(prop *MCMSProposal) ([]byte, error) {
	// Convert the MCMS proposal to a V1 manifest
	m := manifest.ProposalV1{
		ObjectMeta: manifest.BaseProposal{
			Version:     prop.Version,
			Description: prop.Description,
			Kind:        prop.Kind,
		},
		ObjectConfig: manifest.SpecConfig{
			ValidUntil:           prop.ValidUntil,
			ChainMetadata:        make(map[uint64]manifest.ChainMetadata),
			OverridePreviousRoot: prop.OverridePreviousRoot,
		},
		Signatures:   make([]manifest.Signature, 0, len(prop.Signatures)),
		Transactions: make([]manifest.Transaction, 0, len(prop.Transactions)),
	}

	// Generate the signatures
	for _, sig := range prop.Signatures {
		m.Signatures = append(m.Signatures, manifest.Signature{
			R: sig.R,
			S: sig.S,
			V: sig.V,
		})
	}

	// Map the chain metadata to the chain identifiers
	for sel, md := range prop.ChainMetadata {
		m.ChainMetadata[uint64(sel)] = manifest.ChainMetadata{
			StartingOpCount: md.StartingOpCount,
			MCMAddress:      md.MCMAddress.Hex(),
		}
	}

	// Map the chain operations to the transactions
	for _, t := range prop.Transactions {
		tx := manifest.Transaction{
			ChainSelector:    uint64(t.ChainIdentifier),
			To:               t.To.String(),
			Data:             t.Data,
			AdditionalFields: t.AdditionalFields,
			ContractType:     t.ContractType,
			Tags:             t.Tags,
		}

		m.Transactions = append(m.Transactions, tx)
	}

	return json.Marshal(m)
}
