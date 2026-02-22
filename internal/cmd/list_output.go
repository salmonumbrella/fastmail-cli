package cmd

import "fmt"

func printList(header string, items []string) {
	fmt.Println(header)
	for _, item := range items {
		fmt.Printf("  - %s\n", item)
	}
}
