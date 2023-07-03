package nodep

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type vmessQrCode struct {
	Ps   string      `json:"ps,omitempty"`
	Add  string      `json:"add,omitempty"`
	Port interface{} `json:"port,omitempty"`
	Id   string      `json:"id,omitempty"`
	Scy  string      `json:"scy,omitempty"`
	Net  string      `json:"net,omitempty"`
	Type string      `json:"type,omitempty"`
	Host string      `json:"host,omitempty"`
	Path string      `json:"path,omitempty"`
	Tls  string      `json:"tls,omitempty"`
	Sni  string      `json:"sni,omitempty"`
	Alpn string      `json:"alpn,omitempty"`
	Fp   string      `json:"fp,omitempty"`
}

func parseVMessQrCode(text string) (*XrayOutbound, error) {
	qrcodeBytes := []byte(text)
	qrcode := vmessQrCode{}

	err := json.Unmarshal(qrcodeBytes, &qrcode)
	if err != nil {
		return nil, err
	}

	return qrcode.outbound()
}

func (proxy vmessQrCode) outbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "vmess"
	outbound.Name = proxy.Ps

	var user XrayVMessVnextUser
	user.Id = proxy.Id
	user.Security = proxy.Scy

	var vnext XrayVMessVnext
	vnext.Address = proxy.Add
	portStr := fmt.Sprintf("%v", proxy.Port)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	vnext.Port = port

	vnext.Users = []XrayVMessVnextUser{user}

	var settings XrayVMess
	settings.Vnext = []XrayVMessVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings()

	return &outbound, nil
}

func (proxy vmessQrCode) streamSettings() *XrayStreamSettings {
	var streamSettings XrayStreamSettings
	network := proxy.Net
	if len(network) == 0 {
		streamSettings.Network = "tcp"
	} else {
		streamSettings.Network = network
	}

	switch streamSettings.Network {
	case "tcp":
		headerType := proxy.Type
		if headerType == "http" {
			var request XrayTcpSettingsHeaderRequest
			path := proxy.Path
			if len(path) > 0 {
				request.Path = strings.Split(path, ",")
			}
			host := proxy.Host
			if len(host) > 0 {
				var headers XrayTcpSettingsHeaderRequestHeaders
				headers.Host = strings.Split(host, ",")
				request.Headers = &headers
			}
			var header XrayTcpSettingsHeader
			header.Type = headerType
			header.Request = &request

			var tcpSettings XrayTcpSettings
			tcpSettings.Header = &header

			streamSettings.TcpSettings = &tcpSettings
		}
	case "kcp":
		var kcpSettings XrayKcpSettings
		headerType := proxy.Type
		if len(headerType) > 0 {
			var header XrayFakeHeader
			header.Type = headerType
			kcpSettings.Header = &header
		}
		kcpSettings.Seed = proxy.Path

		streamSettings.KcpSettings = &kcpSettings
	case "ws":
		var wsSettings XrayWsSettings
		wsSettings.Path = proxy.Path
		host := proxy.Host
		if len(host) > 0 {
			var headers XrayWsSettingsHeaders
			headers.Host = host
			wsSettings.Headers = &headers
		}

		streamSettings.WsSettings = &wsSettings
	case "grpc":
		var grcpSettings XrayGrpcSettings
		grcpSettings.ServiceName = proxy.Path
		mode := proxy.Type
		grcpSettings.MultiMode = mode == "multi"

		streamSettings.GrpcSettings = &grcpSettings
	case "quic":
		var quicSettings XrayQuicSettings
		headerType := proxy.Type
		if len(headerType) > 0 {
			var header XrayFakeHeader
			header.Type = headerType
			quicSettings.Header = &header
		}
		quicSettings.Security = proxy.Host
		quicSettings.Key = proxy.Path

		streamSettings.QuicSettings = &quicSettings
	case "http":
		var httpSettings XrayHttpSettings
		host := proxy.Host
		httpSettings.Host = strings.Split(host, ",")
		httpSettings.Path = proxy.Path

		streamSettings.HttpSettings = &httpSettings
	}

	proxy.parseSecurity(&streamSettings)

	return &streamSettings
}

func (proxy vmessQrCode) parseSecurity(streamSettings *XrayStreamSettings) {
	var tlsSettings XrayTlsSettings

	tlsSettings.Fingerprint = proxy.Fp
	tlsSettings.ServerName = proxy.Sni

	alpn := proxy.Alpn
	if len(alpn) > 0 {
		tlsSettings.Alpn = strings.Split(alpn, ",")
	}

	security := proxy.Tls
	if len(security) == 0 {
		streamSettings.Security = "none"
	} else {
		streamSettings.Security = security
	}

	// some link omits too many params, here is some fixing
	if streamSettings.Network == "ws" && len(tlsSettings.ServerName) == 0 {
		if streamSettings.WsSettings != nil && streamSettings.WsSettings.Headers != nil {
			tlsSettings.ServerName = streamSettings.WsSettings.Headers.Host
		}
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TlsSettings = &tlsSettings
	}
}
