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

package testing

import (
	"net"
	"time"

	"github.com/whtsky/gocat/relay"
)

type TCPClient struct {
	connection net.Conn
}

func NewTCPClient(address string) (*TCPClient, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &TCPClient{
		connection: relay.NewDeadlineConnection(conn, 30*time.Second, 30*time.Second),
	}, nil
}

func (c *TCPClient) SendMsg(msg []byte) (int, error) {
	return c.connection.Write(msg)
}

func (c *TCPClient) ReceiveMsg(bufferSize int) ([]byte, error) {
	offset := 0
	buf := make([]byte, bufferSize, bufferSize+1)

	for offset < len(buf) {
		n, err := c.connection.Read(buf[offset:])
		if err != nil {
			return nil, err
		}

		offset += n
	}

	return buf, nil
}

func (c *TCPClient) Close() {
	_ = c.connection.Close()
}
