package grokstaterrors

import "errors"

var (
	OK = errors.New("OK.")

	NoDefaultConfig          = errors.New("Default config file not found.")
	ErrorLoadingCustomConfig = errors.New("Error loading custom config file.")

	IPv6NotSupported = errors.New("IPv6 is not supported yet.")

	NoProtocol = errors.New("Please specify the protocol.")
	NoHosts    = errors.New("Please specify the hosts to query.")

	InvalidProtocol = errors.New("Invalid protocol specified.")
	InvalidMasterOf = errors.New("Invalid query part attached to master protocol.")

	ServerDown = errors.New("Server down.")

	MalformedPacket = errors.New("Malformed packet.")

	InvalidResponsePrelude   = errors.New("Invalid response prelude.")
	InvalidResponseLength    = errors.New("Invalid response length.")
	InvalidResponseChallenge = errors.New("Invalid response challenge.")

	InvalidServerEntryInMasterResponse = errors.New("Invalid server entry in the master server response.")

	NoInfoResponse    = errors.New("No info response.")
	NoServersResponse = errors.New("No servers response.")
	NoStatusResponse  = errors.New("No status response.")

	InvalidServerHeader = errors.New("Invalid server header.")

	InvalidPlayerString       = errors.New("Invalid player string.")
	InvalidPlayerStringLength = errors.New("Invalid player string length.")

	InvalidRuleString       = errors.New("Invalid rule string.")
	InvalidRuleStringLength = errors.New("Invalid rule string length.")

	CompError = errors.New("Mismatch.")
)
