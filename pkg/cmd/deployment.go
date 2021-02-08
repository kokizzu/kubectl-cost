package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/spf13/cobra"
)

type CostOptionsDeployment struct {
	isHistorical    bool
	filterNamespace string

	displayOptions
}

func newCmdCostDeployment(streams genericclioptions.IOStreams) *cobra.Command {
	commonO := NewCommonCostOptions(streams)
	deploymentO := &CostOptionsDeployment{}

	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "view cost information aggregated by deployment",
		RunE: func(c *cobra.Command, args []string) error {
			if err := commonO.Complete(c, args); err != nil {
				return err
			}
			if err := commonO.Validate(); err != nil {
				return err
			}

			return runCostDeployment(commonO, deploymentO)
		},
	}

	cmd.Flags().StringVar(&commonO.costWindow, "window", "yesterday", "the window of data to query")
	cmd.Flags().BoolVar(&deploymentO.isHistorical, "historical", false, "show the total cost during the window instead of the projected monthly rate based on the data in the window")
	cmd.Flags().BoolVar(&deploymentO.showCPUCost, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&deploymentO.showMemoryCost, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&deploymentO.showGPUCost, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&deploymentO.showPVCost, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&deploymentO.showNetworkCost, "show-network", false, "show data for network cost")
	cmd.Flags().BoolVar(&deploymentO.showEfficiency, "show-efficiency", false, "show efficiency of cost alongside CPU and memory cost. Only works with --historical.")
	cmd.Flags().StringVarP(&deploymentO.filterNamespace, "namespace-filter", "N", "", "Limit results to only one namespace. Defaults to all namespaces.")
	commonO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostDeployment(co *CostOptionsCommon, no *CostOptionsDeployment) error {

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if !no.isHistorical {
		aggCMResp, err := queryAggCostModel(clientset, co.costWindow, "deployment")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		// don't show unallocated deployment data
		delete(aggCMResp.Data, "__unallocated__")

		applyNamespaceFilter(aggCMResp.Data, no.filterNamespace)

		err = writeAggregationRateTable(
			co.Out,
			aggCMResp.Data,
			[]string{"namespace", "deployment"},
			deploymentTitleExtractor,
			no.displayOptions,
		)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		// Not supported because the allocation API does not return deployment names.
		return fmt.Errorf("kubectl cost deployment does not yet support historical queries")
	}

	return nil
}

// Applies the filter in place by deleting all entries from aggData that are not in
// the namespace, unless it is an empty string in which case nothing is done.
func applyNamespaceFilter(aggData map[string]aggregation, namespaceFilter string) {
	if namespaceFilter == "" {
		return
	}

	for aggName, _ := range aggData {
		sp, err := deploymentTitleExtractor(aggName)
		if err != nil {
			klog.Warningf("failed to extract namespace info from aggregation title %s, skipping", aggName)
			continue
		}
		namespace := sp[0]

		if namespace != namespaceFilter {
			delete(aggData, aggName)
		}
	}

	return
}