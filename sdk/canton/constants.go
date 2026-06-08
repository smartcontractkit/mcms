package canton

const (
	MCMSTemplateKey = "MCMS.Main:MCMS"

	rawDataKeyNewMCMSContractID = "NewMCMSContractID"
	rawDataKeyNewMCMSTemplateID = "NewMCMSTemplateID"
	rawDataKeyRawTx             = "RawTx"

	instanceAddressHexLen = 64
	hexWordLen            = 64
	templateIDPartCount   = 3
	hexEncodedByteLen     = 2
	maxMCMSGroups         = 32
	microsecondsPerSecond = 1_000_000

	defaultCantonChainID int64 = 1

	mcmsInstanceIDCCIP    = "mcms-ccip"
	mcmsInstanceIDCCV     = "mcms-ccv"
	mcmsInstanceIDDefault = "mcms"
)
