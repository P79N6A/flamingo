package kiterrno

// error code
const (
	SuccessCode   = 0 // SuccessCode Success
	UserErrorCode = -1

	// error codes for circuitbreaker MW
	NotAllowedByServiceCBCode  = 101
	NotAllowedByInstanceCBCode = 102
	RPCTimeoutCode             = 103

	// error codes for degradation MW
	ForbiddenByDegradationCode     = 104
	GetDegradationPercentErrorCode = 105

	// error codes for conn retry MW
	BadConnBalancerCode = 106
	BadConnRetrierCode  = 107
	ConnRetryCode       = 108

	// error codes for loanrpc retry
	BadRPCRetrierCode = 109
	RPCRetryCode      = 110

	// error codes for common
	NoExpectedFieldCode = 111 // NoExpectedFieldCode will be returned if there is no expected field in the context

	// error codes for pool
	GetConnErrorCode = 112

	// error codes for service discover
	ServiceDiscoverCode = 113

	// error codes for IDC selector
	IDCSelectErrorCode = 114

	// error codes for ACL
	NotAllowedByACLCode = 115

	// error codes for network I/O
	ReadTimeoutCode     = 116
	WriteTimeoutCode    = 117
	ConnResetByPeerCode = 118

	// error describes
	NotAllowedByServiceCBDesc      = "Not allowed by service circuitbreaker"
	NotAllowedByInstanceCBDesc     = "Downstream service's network is bad, not allowed by dialer circuitbreaker"
	RPCTimeoutDesc                 = "RPC timeout"
	ForbiddenByDegradationDesc     = "Forbidden by degradation"
	GetDegradationPercentErrorDesc = "Get degradation percent error"
	BadConnBalancerDesc            = "Create Balancer error"
	BadConnRetrierDesc             = "Create Conn Retrier error"
	ConnRetryDesc                  = "All Conn retries have failed"
	BadRPCRetrierDesc              = "Create RPC Retrier error"
	RPCRetryDesc                   = "All RPC retries have failed"
	NoExpectedFieldDesc            = "No expected field in the context"
	GetConnErrorDesc               = "Get connection error"
	ServiceDiscoverDesc            = "Service discover error"
	IDCSelectErrorDesc             = "Select IDC error"
	NotAllowedByACLDesc            = "Not allowed by ACL"
	ReadTimeoutDesc                = "Read network timeout"
	WriteTimeoutDesc               = "Write network timeout"
	ConnResetByPeerDesc            = "Conn reset by peer"
)

// IsNetErrCode returns if this error is caused by network
func IsNetErrCode(code int) bool {
	switch code {
	case GetConnErrorCode: // WriteConnErrCode, ReadConnErrCode
		return true
	}
	return false
}

// IsKiteErrCode returns if this code is defined and used by kite
func IsKiteErrCode(code int) bool {
	switch code {
	case NotAllowedByServiceCBCode, NotAllowedByInstanceCBCode, ForbiddenByDegradationCode, GetDegradationPercentErrorCode,
		BadConnBalancerCode, BadConnRetrierCode, ConnRetryCode, NoExpectedFieldCode, ServiceDiscoverCode,
		IDCSelectErrorCode, GetConnErrorCode:
		return true
	}
	return false
}
