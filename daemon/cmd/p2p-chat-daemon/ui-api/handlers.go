// daemon/cmd/p2p-chat-daemon/handlers.go
package ui_api // Still part of the executable package

import (
	"encoding/json" // To encode Go data structures into JSON
	"log"
	"net/http" // Needed for request/response writing and status codes

	"github.com/libp2p/go-libp2p/core/host" // Needed to access node information
)

// --- Handler Factory Functions ---
// These functions take dependencies (like the libp2p host) and return
// the actual http.HandlerFunc that will handle requests for a specific route.
// This pattern makes handlers testable and keeps routing clean in server.go.

// idHandler creates an HTTP handler function that returns the libp2p node's Peer ID.
// It takes the Host instance as input (dependency injection).
func idHandler(node host.Host) http.HandlerFunc {
	// Return the actual handler function (a closure)
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS Headers: Crucial for allowing the React UI (running on a different port
		// in the browser) to make requests to this API server.
		// '*' allows any origin. For production, restrict this to your UI's specific origin.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS") // Allowed HTTP methods
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // Allowed headers

		// Browsers often send an OPTIONS request (preflight) before a "complex" request
		// (like one with custom headers or non-simple methods) to check CORS permissions.
		// We need to handle this by just returning OK status.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// --- Actual Handler Logic ---
		// Check if the node object is valid (it should be by this point)
		if node == nil {
			http.Error(w, "Node not initialized", http.StatusInternalServerError)
			return
		}

		// Get the Peer ID from the node.
		id := node.ID()

		// Prepare the JSON response payload. Using a map is common for simple responses.
		response := map[string]string{"peerId": id.String()} // Convert Peer ID to string

		// Set the Content-Type header to indicate we're sending JSON.
		w.Header().Set("Content-Type", "application/json")

		// Encode the 'response' map directly to the ResponseWriter 'w'.
		// json.NewEncoder is efficient for streaming JSON output.
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log errors happening during response writing.
			log.Printf("Error encoding JSON response for /api/id: %v", err)
			// Send a generic server error back to the client if encoding fails.
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// --- Add more handler factory functions here for other endpoints ---
// e.g., func peersHandler(node host.Host) http.HandlerFunc { ... }
//       func sendHandler(node host.Host) http.HandlerFunc { ... }
