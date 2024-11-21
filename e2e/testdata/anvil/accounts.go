package anvil

// KeyPair represents a private-public key pair.
type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

// Accounts holds the predefined accounts for Anvil.
var Accounts = []KeyPair{
	{
		PrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		PublicKey:  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
	},
	{
		PrivateKey: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
		PublicKey:  "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
	},
	{
		PrivateKey: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a",
		PublicKey:  "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
	},
}
