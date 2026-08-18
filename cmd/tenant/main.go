// Command tenant manages Verigate tenants — the practical admin surface
// for multi-tenancy tonight, rather than a full admin HTTP API.
//
// Usage:
//
//	go run ./cmd/tenant create --name "acme corp" --rpm 60
//	go run ./cmd/tenant list
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/rakshit-gen/verigate/internal/config"
	"github.com/rakshit-gen/verigate/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	_ = godotenv.Load()
	cfg := config.Load()
	ctx := context.Background()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to postgres: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	switch os.Args[1] {
	case "create":
		runCreate(ctx, st, os.Args[2:])
	case "list":
		runList(ctx, st)
	default:
		usage()
		os.Exit(1)
	}
}

func runCreate(ctx context.Context, st *store.Store, args []string) {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	name := fs.String("name", "", "tenant name (required)")
	rpm := fs.Int("rpm", 60, "requests per minute this tenant is limited to")
	fs.Parse(args)

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: --name is required")
		os.Exit(1)
	}

	tenant, apiKey, err := st.CreateTenant(ctx, *name, *rpm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create tenant: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created tenant %q (id: %s, rate limit: %d req/min)\n\n", tenant.Name, tenant.ID, tenant.RateLimitRPM)
	fmt.Printf("API key (save this now — it will not be shown again):\n\n  %s\n\n", apiKey)
	fmt.Println("Use it exactly like the static VERIGATE_API_KEY: Authorization: Bearer " + apiKey)
}

func runList(ctx context.Context, st *store.Store) {
	tenants, err := st.ListTenants(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list tenants: %v\n", err)
		os.Exit(1)
	}
	if len(tenants) == 0 {
		fmt.Println("no tenants yet — create one with: go run ./cmd/tenant create --name <name>")
		return
	}
	fmt.Printf("%-38s %-24s %-10s %s\n", "ID", "NAME", "RPM LIMIT", "CREATED")
	for _, t := range tenants {
		fmt.Printf("%-38s %-24s %-10d %s\n", t.ID, t.Name, t.RateLimitRPM, t.CreatedAt.Format("2006-01-02 15:04"))
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/tenant create --name <name> [--rpm 60]")
	fmt.Println("  go run ./cmd/tenant list")
}
