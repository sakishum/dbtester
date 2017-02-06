// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agent

import (
	"net"
	"os"
	"path/filepath"

	"github.com/coreos/dbtester/dbtesterpb"
	"github.com/coreos/dbtester/pkg/netutil"
	"github.com/coreos/dbtester/pkg/ntp"
	"github.com/coreos/pkg/capnslog"
	"github.com/gyuho/psn"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type flags struct {
	agentLog                     string
	databaseLog                  string
	systemMetricsCSV             string
	systemMetricsCSVInterpolated string

	javaExec   string
	etcdExec   string
	zetcdExec  string
	cetcdExec  string
	consulExec string

	zkWorkDir     string
	zkDataDir     string
	zkConfig      string
	etcdDataDir   string
	consulDataDir string

	grpcPort         string
	diskDevice       string
	networkInterface string
	clientNumPath    string
}

var globalFlags flags

func init() {
	dn, err := psn.GetDevice("/")
	if err != nil {
		plog.Warningf("cannot get disk device mounted at '/' (%v)", err)
	}
	nm, err := netutil.GetDefaultInterfaces()
	if err != nil {
		plog.Warningf("cannot detect default network interface (%v)", err)
	}
	var nt string
	for k := range nm {
		nt = k
		break
	}

	Command.PersistentFlags().StringVar(&globalFlags.agentLog, "agent-log", filepath.Join(homeDir(), "agent.log"), "agent log path.")
	Command.PersistentFlags().StringVar(&globalFlags.databaseLog, "database-log", filepath.Join(homeDir(), "database.log"), "Database log path.")
	Command.PersistentFlags().StringVar(&globalFlags.systemMetricsCSV, "system-metrics-csv", filepath.Join(homeDir(), "server-system-metrics.csv"), "Raw system metrics data path.")
	Command.PersistentFlags().StringVar(&globalFlags.systemMetricsCSVInterpolated, "system-metrics-csv-interpolated", filepath.Join(homeDir(), "server-system-metrics-interpolated.csv"), "Interpolated system metrics data path.")

	Command.PersistentFlags().StringVar(&globalFlags.javaExec, "java-exec", "/usr/bin/java", "Java executable binary path (needed for Zookeeper).")
	Command.PersistentFlags().StringVar(&globalFlags.etcdExec, "etcd-exec", filepath.Join(os.Getenv("GOPATH"), "bin/etcd"), "etcd executable binary path.")
	Command.PersistentFlags().StringVar(&globalFlags.zetcdExec, "zetcd-exec", filepath.Join(os.Getenv("GOPATH"), "bin/zetcd"), "zetcd executable binary path .")
	Command.PersistentFlags().StringVar(&globalFlags.cetcdExec, "cetcd-exec", filepath.Join(os.Getenv("GOPATH"), "bin/cetcd"), "cetcd executable binary path .")
	Command.PersistentFlags().StringVar(&globalFlags.consulExec, "consul-exec", filepath.Join(os.Getenv("GOPATH"), "bin/consul"), "Consul executable binary path.")

	Command.PersistentFlags().StringVar(&globalFlags.zkWorkDir, "zookeeper-work-dir", filepath.Join(homeDir(), "zookeeper"), "Zookeeper working directory.")
	Command.PersistentFlags().StringVar(&globalFlags.zkDataDir, "zookeeper-data-dir", filepath.Join(homeDir(), "zookeeper/zookeeper.data"), "Zookeeper data directory.")
	Command.PersistentFlags().StringVar(&globalFlags.zkConfig, "zookeeper-config", filepath.Join(homeDir(), "zookeeper/zookeeper.config"), "Zookeeper configuration file path.")
	Command.PersistentFlags().StringVar(&globalFlags.etcdDataDir, "etcd-data-dir", filepath.Join(homeDir(), "etcd.data"), "etcd data directory.")
	Command.PersistentFlags().StringVar(&globalFlags.consulDataDir, "consul-data-dir", filepath.Join(homeDir(), "consul.data"), "Consul data directory.")

	Command.PersistentFlags().StringVar(&globalFlags.grpcPort, "agent-port", ":3500", "Port to server agent gRPC server.")
	Command.PersistentFlags().StringVar(&globalFlags.diskDevice, "disk-device", dn, "Disk device to collect disk statistics metrics from.")
	Command.PersistentFlags().StringVar(&globalFlags.networkInterface, "network-interface", nt, "Network interface to record in/outgoing packets.")
	Command.PersistentFlags().StringVar(&globalFlags.clientNumPath, "client-num-path", filepath.Join(homeDir(), "client-num"), "File path to store client number.")
}

// Command implements 'agent' command.
var Command = &cobra.Command{
	Use:   "agent",
	Short: "Database 'agent' in remote servers.",
	RunE:  commandFunc,
}

func commandFunc(cmd *cobra.Command, args []string) error {
	no, nerr := ntp.DefaultSync()
	plog.Infof("npt update output: %q", no)
	plog.Infof("npt update error: %v", nerr)

	f, err := openToAppend(globalFlags.agentLog)
	if err != nil {
		return err
	}
	defer f.Close()
	capnslog.SetFormatter(capnslog.NewPrettyFormatter(f, false))

	var (
		grpcServer = grpc.NewServer()
		sender     = NewServer()
	)
	ln, err := net.Listen("tcp", globalFlags.grpcPort)
	if err != nil {
		return err
	}
	dbtesterpb.RegisterTransporterServer(grpcServer, sender)

	plog.Infof("agent started with gRPC %s (log path %q)", globalFlags.grpcPort, globalFlags.agentLog)
	return grpcServer.Serve(ln)
}
