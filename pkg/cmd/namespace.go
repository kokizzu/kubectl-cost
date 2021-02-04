package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

type CostOptionsNamespace struct {
	isRate      bool
	showCPU     bool
	showMemory  bool
	showGPU     bool
	showPV      bool
	showNetwork bool
}

func newCmdCostNamespace(streams genericclioptions.IOStreams) *cobra.Command {
	commonO := NewCommonCostOptions(streams)
	namespaceO := &CostOptionsNamespace{}

	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "view cost information aggregated by namespace",
		RunE: func(c *cobra.Command, args []string) error {
			if err := commonO.Complete(c, args); err != nil {
				return err
			}
			if err := commonO.Validate(); err != nil {
				return err
			}

			return runCostNamespace(commonO, namespaceO)
		},
	}

	cmd.Flags().StringVar(&commonO.costWindow, "window", "yesterday", "the window of data to query")
	cmd.Flags().BoolVar(&namespaceO.isRate, "rate", false, "show the projected monthly rate based on data in the window instead of the total cost during the window")
	cmd.Flags().BoolVar(&namespaceO.showCPU, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&namespaceO.showMemory, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&namespaceO.showGPU, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&namespaceO.showPV, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&namespaceO.showNetwork, "show-network", false, "show data for network cost")
	commonO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostNamespace(co *CostOptionsCommon, no *CostOptionsNamespace) error {

	do := displayOptions{
		showCPUCost:     no.showCPU,
		showMemoryCost:  no.showMemory,
		showGPUCost:     no.showGPU,
		showPVCost:      no.showPV,
		showNetworkCost: no.showNetwork,
	}

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if no.isRate {
		aggCMResp, err := queryAggCostModel(clientset, co.costWindow, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		err = writeAggregationRateTable(
			co.Out,
			aggCMResp.Data,
			[]string{"namespace"},
			noopTitleExtractor,
			do,
		)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		allocR, err := queryAllocation(clientset, co.costWindow, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query allocation API")
		}

		// Use Data[0] because the query accumulates
		err = writeNamespaceTable(co.Out, allocR.Data[0], do)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}
