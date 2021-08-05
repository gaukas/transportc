package s2sHelper

import (
	s2s "github.com/Gaukas/seed2sdp"
)

const (
	ServerClientSharedSecret string = "A VERY LONG AND STRONG SECRET"
	OffererIdentifier        string = "reffO"
	AnswererIdentifier       string = "rewsnA"
)

var malleables = s2s.NewSDPMalleables()
var hkdfOffer = s2s.NewHKDFParams().SetSecret(ServerClientSharedSecret).SetInfoPrefix(OffererIdentifier)
var hkdfAnswer = s2s.NewHKDFParams().SetSecret(ServerClientSharedSecret).SetInfoPrefix(AnswererIdentifier)

func InflateSdpWithSeed(seed string, deflatedSDP s2s.SDPDeflated) (*s2s.SDP, error) {
	var hkdfParams *s2s.HKDFParams
	if deflatedSDP.SDPType == s2s.SDPOffer {
		hkdfParams = hkdfOffer.SetSalt(seed)
	} else if deflatedSDP.SDPType == s2s.SDPAnswer {
		hkdfParams = hkdfAnswer.SetSalt(seed)
	}
	sdp, err := deflatedSDP.Inflate()
	if err != nil {
		return nil, err
	}

	sdp.Malleables = malleables
	sdp.Fingerprint, err = s2s.PredictDTLSFingerprint(hkdfParams)
	if err != nil {
		return nil, err
	}

	sdp.IceParams, err = s2s.PredictIceParameters(hkdfParams)
	if err != nil {
		return nil, err
	}

	return sdp, nil
}
