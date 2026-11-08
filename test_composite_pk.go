package main

import "fmt"
import "strings"

func main() {
	pks := []string{"room_id", "equipment_id"}
	fmt.Printf("model.JoinKey(%s)", strings.Join(pks, ", "))
}
