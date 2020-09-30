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

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/palantir/stacktrace"
	"github.com/spf13/cobra"

	"github.com/whtsky/gocat/relay"
)

func NewUnixToTCPCmd(logger relay.Logger) *cobra.Command {
	var unixToTCPSocketPath string
	var unixToTCPAddressPath string
	var bufferSize int

	cmdInstance := &cobra.Command{
		Use:   "unix-to-tcp",
		Short: "relay from a unix source to tcp clients",
		Long:  `relay from a unix source to tcp clients`,
		RunE: func(command *cobra.Command, args []string) error {
			// nolint: gocritic
			if len(unixToTCPSocketPath) < 0 {
				return stacktrace.NewError("blank/empty `src` specified")
			}

			// nolint: gocritic
			if len(unixToTCPAddressPath) < 0 {
				return stacktrace.NewError("blank/empty `dst` specified")
			}

			relayer, err := relay.NewUnixSocketTCP(
				logger,
				unixToTCPSocketPath,
				unixToTCPAddressPath,
				bufferSize,
			)
			if err != nil {
				return stacktrace.Propagate(err, "couldn't create relay from unix socket to TCP")
			}

			osSignalCh := make(chan os.Signal, 1)
			defer close(osSignalCh)

			signal.Notify(osSignalCh, os.Interrupt, syscall.SIGTERM)

			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()

			// Ctrl+C handler
			go func() {
				<-osSignalCh
				signal.Stop(osSignalCh)
				cancelFunc()
			}()

			err = relayer.Relay(ctx)
			return stacktrace.Propagate(err, "couldn't relay from unix socket to TCP")
		},
	}

	cmdInstance.Flags().StringVar(
		&unixToTCPSocketPath,
		"src",
		"",
		"source of unix domain socket",
	)
	_ = cmdInstance.MarkFlagRequired("src")
	cmdInstance.Flags().StringVar(
		&unixToTCPAddressPath,
		"dst",
		"",
		"destination to TCP listen",
	)
	_ = cmdInstance.MarkFlagRequired("dst")
	cmdInstance.Flags().IntVar(
		&bufferSize,
		"buffer-size",
		DefaultBufferSize,
		"Buffer size in bytes of the data stream",
	)

	return cmdInstance
}
