package ctx

type contextValue string

var (
	vaultClient       contextValue = "ctx:vc"
	templateMap       contextValue = "ctx:tm"
	credentialReqUnit contextValue = "ctx:cru"
	credentialReqCred contextValue = "ctx:crc"
	socketPath        contextValue = "ctx:sp"
	globalDebug       contextValue = "ctx:gd"
)
