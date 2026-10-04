package main
import (
	"fmt"
	"github.com/gork-labs/gork/pkg/api"
)
func main(){
	e := api.NewDocExtractor()
	_ = e.ParseExternalModule("github.com/stripe/stripe-go/v76")
	d := e.ExtractTypeDoc("Event")
	fmt.Printf("Type desc: %q, fields=%d\n", d.Description, len(d.Fields))
	for k,v := range d.Fields { if v.Description!="" { fmt.Printf("%s => %s\n", k, v.Description) } }
}
