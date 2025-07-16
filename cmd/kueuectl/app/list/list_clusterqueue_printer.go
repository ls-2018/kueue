package list

import (
	"errors"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

type listClusterQueuePrinter struct {
	clock        clock.Clock
	printOptions printers.PrintOptions
}

var _ printers.ResourcePrinter = (*listClusterQueuePrinter)(nil)

func (p *listClusterQueuePrinter) PrintObj(obj runtime.Object, out io.Writer) error {
	printer := printers.NewTablePrinter(p.printOptions)

	list, ok := obj.(*v1beta1.ClusterQueueList)
	if !ok {
		return errors.New("invalid object type")
	}

	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name"},
			{Name: "Cohort", Type: "string"},
			{Name: "Pending Workloads", Type: "integer"},
			{Name: "Admitted Workloads", Type: "integer"},
			{Name: "Active", Type: "boolean"},
			{Name: "Age", Type: "string"},
		},
		Rows: p.printClusterQueueList(list),
	}

	return printer.PrintObj(table, out)
}

func (p *listClusterQueuePrinter) WithHeaders(f bool) *listClusterQueuePrinter {
	p.printOptions.NoHeaders = !f
	return p
}

func (p *listClusterQueuePrinter) WithClock(c clock.Clock) *listClusterQueuePrinter {
	p.clock = c
	return p
}

func newClusterQueueTablePrinter() *listClusterQueuePrinter {
	return &listClusterQueuePrinter{
		clock: clock.RealClock{},
	}
}

func (p *listClusterQueuePrinter) printClusterQueueList(list *v1beta1.ClusterQueueList) []metav1.TableRow {
	rows := make([]metav1.TableRow, len(list.Items))
	for index := range list.Items {
		rows[index] = p.printClusterQueue(&list.Items[index])
	}
	return rows
}

func (p *listClusterQueuePrinter) printClusterQueue(clusterQueue *v1beta1.ClusterQueue) metav1.TableRow {
	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: clusterQueue},
	}
	row.Cells = []any{
		clusterQueue.Name,
		clusterQueue.Spec.Cohort,
		clusterQueue.Status.PendingWorkloads,
		clusterQueue.Status.AdmittedWorkloads,
		isActiveStatus(clusterQueue),
		duration.HumanDuration(p.clock.Since(clusterQueue.CreationTimestamp.Time)),
	}

	return row
}
