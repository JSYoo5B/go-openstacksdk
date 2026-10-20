// Command paritycheck validates durable SDK support reviews against the pinned,
// declared API inventories. An unreviewed operation remains unresolved.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	root := flag.String("root", ".", "SDK module root")
	sync := flag.Bool("sync", false, "refresh the operation catalog without changing reviews")
	flag.Parse()
	counts, err := check(*root, *sync)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Declared inventory: %d operations; supported=%d go_mapping=%d unsupported=%d unresolved=%d\n",
		counts.total, counts.status["supported"], counts.status["go_mapping"], counts.status["unsupported"], counts.status["unresolved"])
	fmt.Println("Scope: declared inventory only; inherited, descriptor and Resource surfaces still need separate review.")
}
