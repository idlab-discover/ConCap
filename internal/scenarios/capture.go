package scenarios

import (
	"context"
	"fmt"
	"log"

	kubeapi "github.com/idlab-discover/concap/internal/kubernetes"
)

const (
	RawPcapPath        = DataMountPath + "/dump.raw.pcap"
	NormalizedPcapPath = DataMountPath + "/dump.pcap"
	TcpdumpLogPath     = DataMountPath + "/tcpdump.log"
	TcpdumpPidPath     = DataMountPath + "/tcpdump.pid"
	ReordercapLogPath  = DataMountPath + "/reordercap.log"
)

func startTcpdumpCapture(ctx context.Context, podName, filter string) error {
	stdo, stde, err := kubeapi.ExecShellInContainer(
		ctx,
		kubeapi.WorkloadNamespace,
		podName,
		TcpdumpContainerName,
		`nohup tcpdump --no-promiscuous-mode --immediate-mode --buffer-size=32768 --packet-buffered -n --interface=eth0 -w `+RawPcapPath+` "`+filter+`" > `+TcpdumpLogPath+` 2>&1 & echo $! > `+TcpdumpPidPath,
	)
	if err != nil {
		return err
	}
	if stde != "" {
		log.Printf("tcpdump start stdout: %s\n\tstderr: %s", stdo, stde)
	}
	return nil
}

func stopAndNormalizeCapture(ctx context.Context, podName string) error {
	if err := stopTcpdumpCapture(ctx, podName); err != nil {
		return err
	}
	if err := reorderCapture(ctx, podName); err != nil {
		return err
	}
	return nil
}

func stopTcpdumpCapture(ctx context.Context, podName string) error {
	_, _, err := kubeapi.ExecShellInContainer(
		ctx,
		kubeapi.WorkloadNamespace,
		podName,
		TcpdumpContainerName,
		`kill -SIGINT $(cat `+TcpdumpPidPath+`) &&
		 while ! ps | grep "\[tcpdump\]" 2>/dev/null; do
			 sleep 1;
		 done`,
	)
	if err != nil {
		return fmt.Errorf("stop tcpdump: %w", err)
	}
	return nil
}

func reorderCapture(ctx context.Context, podName string) error {
	stdo, stde, err := kubeapi.ExecShellInContainer(
		ctx,
		kubeapi.WorkloadNamespace,
		podName,
		ReordercapContainerName,
		`rm -f `+NormalizedPcapPath+` `+ReordercapLogPath+` &&
		 reordercap `+RawPcapPath+` `+NormalizedPcapPath+` > `+ReordercapLogPath+` 2>&1
		 status=$?
		 cat `+ReordercapLogPath+`
		 exit $status`,
	)
	if stdo != "" || stde != "" {
		log.Printf("reordercap stdout: %s\n\tstderr: %s", stdo, stde)
	}
	if err != nil {
		return fmt.Errorf("reorder capture: %w", err)
	}
	return nil
}
