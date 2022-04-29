/******************************************************************************
 * arc_test.go
 * Author:
 * Usage:    `go test`  or  `go test -v`
 * Description:
 *    An incomplete unit testing suite for arc.go. You are welcome to change
 *    anything in this file however you would like. You are strongly encouraged
 *    to create additional tests for your implementation, as the ones provided
 *    here are extremely basic, and intended only to demonstrate how to test
 *    your program.
 ******************************************************************************/

package cache

import (
	"fmt"
	"testing"
)

/******************************************************************************/
/*                                Constants                                   */
/******************************************************************************/
// Constants can go here

/******************************************************************************/
/*                                  Tests                                     */
/******************************************************************************/

func TestARC(t *testing.T) {
	fmt.Println("ARC TESTING")
	capacity := 100
	arc_new := NewARC(capacity)

	// capacity := 64
	// lru := NewLru(capacity)
	// checkCapacity(t, lru, capacity)

	val := []byte("____0")
	arc_new.Set("____0", val)
	val = []byte("____1")
	arc_new.Set("____1", val)
	val = []byte("____2")
	arc_new.Set("____2", val)
	val = []byte("____3")
	arc_new.Set("____3", val)
	val = []byte("____4")
	arc_new.Set("____4", val)
	val = []byte("____5")
	arc_new.Set("____5", val)
	val = []byte("____6")
	arc_new.Set("____6", val)

	val = []byte("____7")
	arc_new.Set("____7", val)

	val = []byte("____8")
	arc_new.Set("____8", val)

	val = []byte("____9")
	arc_new.Set("____9", val)

	// 	if !ok {
	// 		t.Errorf("Failed to add binding with key: %s", key)
	// 		t.FailNow()
	// 	}

	// 	res, _ := lru.Get(key)
	// 	if !bytesEqual(res, val) {
	// 		t.Errorf("Wrong value %s for binding with key: %s", res, key)
	// 		t.FailNow()
	// 	}
	// }

	// val := []byte("12345")
	// lru_new.Set("12345", val)
	fmt.Println("used bytes: ", arc_new.usedBytes)

	fmt.Println("remaining bytes: ", arc_new.RemainingStorage())

	// val = []byte("0")
	arc_new.Get("____0")
	// val = []byte("9")
	val = []byte("___10")
	arc_new.Set("___10", val)
	// fmt.Println("len: ", lru_new.Len())
	fmt.Println("remaining bytes: ", arc_new.RemainingStorage())
	fmt.Println(arc_new.Get("____1"))
	fmt.Println(arc_new.cache)
}
