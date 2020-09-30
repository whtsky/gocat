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
	"io"
	"net"
	"sync"

	"golang.org/x/xerrors"
)

type AbstractDuplexRelay struct {
	logger           Logger
	sourceName       string
	destinationName  string
	destinationAddr  string
	bufferSize       int
	dialSourceConn   func(context.Context) (net.Conn, error)
	listenTargetConn func(context.Context) (net.Listener, error)
}

func (r *AbstractDuplexRelay) Relay(ctx context.Context) error {
	listener, err := r.listenTargetConn(ctx)
	if err != nil {
		return xerrors.Errorf("could bind to %s %s: %w", r.destinationName, r.destinationAddr, err)
	}
	defer listener.Close()

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// NOTE: Don't print false-positive errors
			if ctx.Err() != nil {
				return nil
			}

			continue
		}

		r.logger.Infof("Established connection to %s", conn.RemoteAddr())
		go r.handleConnection(ctx, conn)
	}
}

// nolint:funlen
func (r *AbstractDuplexRelay) handleConnection(ctx context.Context, conn net.Conn) {
	defer func(conn net.Conn) {
		_ = conn.Close()
		r.logger.Infof("Closed connection to %s %s", r.destinationName, conn.RemoteAddr())
	}(conn)

	// NOTE: Accepted connection at `dst` address
	// must be using read/write deadlines to make sure
	// we're not leaking goroutines by waiting on half-closed connections.
	destDeadlineConn := NewDeadlineConnection(conn, writeDeadlineTimeout, readDeadlineTimeout)

	r.logger.Infof("Handling connection from %s %s", r.destinationName, destDeadlineConn.remoteAddress)

	sourceConn, err := r.dialSourceConn(ctx)
	if err != nil {
		r.logger.Errorf(
			"Could not read from source %s. Error: %s",
			r.sourceName,
			err,
		)
		return
	}

	defer sourceConn.Close()
	defer destDeadlineConn.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	// NOTE: Read from source and write to destination
	go func() {
		defer wg.Done()

		buffer := make([]byte, r.bufferSize)

		for {
			readBytes, err := sourceConn.Read(buffer)
			if err != nil {
				sourceConn.Close()
				// NOTE: Force close destination connection to stop
				// the "destination read to source write" goroutine.
				destDeadlineConn.Close()

				if err == io.EOF {
					r.logger.Debugf(
						"Reached EOF of %s %s. Stopping reading",
						r.sourceName,
						sourceConn.RemoteAddr(),
					)
					return
				}

				r.logger.Debugf(
					"Could not read from %s %s. Error: %s\n",
					r.sourceName,
					sourceConn.RemoteAddr(),
					err,
				)
				return
			}

			if readBytes < 1 {
				continue
			}

			// NOTE: Pad to the read bytes to remove 0s
			_, _ = destDeadlineConn.Write(buffer[:readBytes])
		}
	}()

	// NOTE: Read from destination and write to source
	buffer := make([]byte, r.bufferSize)
	for {
		readBytes, err := destDeadlineConn.Read(buffer)
		if err != nil {
			destDeadlineConn.Close()
			// NOTE: Force close source connection to stop
			// the "source read to dest write" goroutine.
			sourceConn.Close()

			if err == io.EOF {
				r.logger.Debugf(
					"Reached EOF of %s %s. Stopping reading",
					r.destinationName,
					destDeadlineConn.remoteAddress,
				)
				break
			}

			r.logger.Debugf(
				"Could not read from %s %s. Error: %s",
				r.destinationName,
				destDeadlineConn.remoteAddress,
				err,
			)
			break
		}

		if readBytes < 1 {
			continue
		}

		// NOTE: Pad to the read bytes to remove 0s
		_, err = sourceConn.Write(buffer[:readBytes])
		if err != nil {
			r.logger.Errorf(
				"Could not write to %s %s. Error: %s",
				r.sourceName,
				sourceConn.RemoteAddr(),
				err,
			)
			return
		}
	}

	wg.Wait()
}
