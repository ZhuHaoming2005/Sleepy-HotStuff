package hacss

import (
	"crypto/rand"
	"fmt"
	"go.dedis.ch/kyber/v3"
	"sleepy-hotstuff/src/cryptolib"
	"math/big"
)

// Define a cyclic group G
type CyclicGroup struct {
	Generators [](*big.Int)
}

type common_reference_string struct {
	CyclicGroup           []*kyber.Group
	Share_Poly_Commitment []*cryptolib.PubPoly
	h_order               int
}

// Generate a random cyclic group
func GenerateCyclicGroup(dim int) *CyclicGroup {
	group := new(CyclicGroup)
	group.Generators = make([](*big.Int), dim)

	// Generate dim random generators
	for i := 0; i < dim; i++ {
		// For simplicity, using big.Int, you may need to adjust based on your requirements
		randomGenerator, err := rand.Int(rand.Reader, big.NewInt(1000000))
		if err != nil {
			fmt.Println("Error generating random number:", err)
			return nil
		}
		group.Generators[i] = randomGenerator
	}

	return group
}

func createCRS() {
	dimension := 3 // Set the dimension of the cyclic group

	// Generate a cyclic group with random generators
	crs := GenerateCyclicGroup(dimension)

	// Print the generated cyclic group
	fmt.Printf("Cyclic Group G:\nGenerators: %v\n", crs.Generators)
}
