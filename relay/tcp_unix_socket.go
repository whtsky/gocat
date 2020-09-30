// Copyright 2018 SumUp Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package relay

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-reuseport"
	"golang.org/x/xerrors"
)

const tcpKeepAlivePeriod = 15 * time.Second

type TCPtoUnixsocket struct {
	AbstractDuplexRelay
}

func NewTCPtoUnixSocket(
	logger Logger,
	tcpAddress,
	unixSocketPath string,
	bufferSize int,
) (*TCPtoUnixsocket, error) {
	tcpAddressParts := strings.Split(tcpAddress, ":")
	if len(tcpAddressParts) != 2 {
		return nil, xerrors.Errorf(
			"wrong format for tcp address %s. Expected <addr>:<port>",
			tcpAddress,
		)
	}

	_, err := strconv.ParseInt(tcpAddressParts[1], 10, 32)
	if err != nil {
		return nil, xerrors.Errorf(
			"could not parse specified port number %s: %w",
			tcpAddressParts[1],
			err,
		)
	}

	return &TCPtoUnixsocket{
		AbstractDuplexRelay{
			logger:          logger,
			sourceName:      "TCP connection",
			destinationName: "unix socket",
			destinationAddr: unixSocketPath,
			bufferSize:      bufferSize,
			dialSourceConn: func(ctx context.Context) (net.Conn, error) {
				dialer := &net.Dialer{
					KeepAlive: tcpKeepAlivePeriod,
				}
				conn, err := dialer.DialContext(
					ctx,
					"tcp",
					tcpAddress,
				)
				if err != nil {
					return nil, xerrors.Errorf(
						"failed to dial TCP address: %s: %w",
						tcpAddress,
						err,
					)
				}

				tcpConn := conn.(*net.TCPConn)
				// TODO: Re-evaluate if this is redundant when `KeepAlive` and `net.Dialer` is used.
				_ = tcpConn.SetKeepAlive(true)
				_ = tcpConn.SetKeepAlivePeriod(tcpKeepAlivePeriod)
				return tcpConn, nil
			},
			listenTargetConn: func(ctx context.Context) (net.Listener, error) {
				// NOTE: This is a streaming unix domain socket
				// equivalent of `sock.STREAM`.
				lc := net.ListenConfig{
					Control: reuseport.Control,
				}
				listener, err := lc.Listen(ctx, "unix", unixSocketPath)
				if err != nil {
					return nil, xerrors.Errorf(
						"failed to listen at Unix socket path %s: %w",
						unixSocketPath,
						err,
					)
				}
				return listener, nil
			},
		},
	}, nil
}
