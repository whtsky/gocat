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
	"os"
	"strconv"
	"strings"

	"github.com/libp2p/go-reuseport"
	"golang.org/x/xerrors"
)

type UnixSocketTCP struct {
	AbstractDuplexRelay
}

func NewUnixSocketTCP(
	logger Logger,
	unixSocketPath,
	tcpAddress string,
	bufferSize int,
) (*UnixSocketTCP, error) {
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

	_, err = os.Stat(unixSocketPath)
	if os.IsNotExist(err) {
		return nil, xerrors.Errorf(
			"could not stat %s: %w",
			unixSocketPath,
			err,
		)
	}

	return &UnixSocketTCP{
		AbstractDuplexRelay{
			logger:          logger,
			bufferSize:      bufferSize,
			sourceName:      "unix socket",
			destinationName: "TCP connection",
			destinationAddr: tcpAddress,
			dialSourceConn: func(ctx context.Context) (net.Conn, error) {
				dialer := &net.Dialer{}
				// NOTE: This is a streaming unix domain socket
				// equivalent of `sock.STREAM`.
				conn, err := dialer.DialContext(ctx, "unix", unixSocketPath)
				if err != nil {
					return nil, xerrors.Errorf(
						"failed to dial unix address %s: %w",
						unixSocketPath,
						err,
					)
				}

				return conn, nil
			},
			listenTargetConn: func(ctx context.Context) (net.Listener, error) {
				lc := net.ListenConfig{
					Control: reuseport.Control,
				}
				listener, err := lc.Listen(ctx, "tcp", tcpAddress)
				if err != nil {
					return nil, xerrors.Errorf(
						"failed to listen at TCP address %s: %w",
						tcpAddress,
						err,
					)
				}
				return listener, nil
			},
		},
	}, nil
}
