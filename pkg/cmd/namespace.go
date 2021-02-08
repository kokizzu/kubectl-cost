package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/spf13/cobra"
)

type CostOptionsNamespace struct {
	isHistorical bool
	showAll      bool

	displayOptions
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

			namespaceO.Complete()

			return runCostNamespace(commonO, namespaceO)
		},
	}

	cmd.Flags().StringVar(&commonO.costWindow, "window", "yesterday", "the window of data to query")
	cmd.Flags().BoolVar(&namespaceO.isHistorical, "historical", false, "show the total cost during the window instead of the projected monthly rate based on the data in the window")
	cmd.Flags().BoolVar(&namespaceO.showCPUCost, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&namespaceO.showMemoryCost, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&namespaceO.showGPUCost, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&namespaceO.showPVCost, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&namespaceO.showNetworkCost, "show-network", false, "show data for network cost")
	cmd.Flags().BoolVar(&namespaceO.showEfficiency, "show-efficiency", false, "Show efficiency of cost alongside CPU and memory cost. Only works with --historical.")
	cmd.Flags().BoolVarP(&namespaceO.showAll, "show-all-resources", "A", false, "Equivalent to --show-cpu --show-memory --show-gpu --show-pv --show-network.")
	commonO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (no *CostOptionsNamespace) Complete() {
	if no.showAll {
		no.showCPUCost = true
		no.showMemoryCost = true
		no.showGPUCost = true
		no.showPVCost = true
		no.showNetworkCost = true
	}
}

func runCostNamespace(co *CostOptionsCommon, no *CostOptionsNamespace) error {

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if !no.isHistorical {
		aggCMResp, err := queryAggCostModel(clientset, co.costWindow, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		err = writeAggregationRateTable(
			co.Out,
			aggCMResp.Data,
			[]string{"namespace"},
			noopTitleExtractor,
			no.displayOptions,
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
		err = writeNamespaceTable(co.Out, allocR.Data[0], no.displayOptions)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}
