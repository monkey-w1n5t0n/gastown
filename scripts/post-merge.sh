#!/bin/bash
# Post-merge hook: Rebuild and install Go binary after merges

cd "$(git rev-parse --show-toplevel)"
echo "🔨 Rebuilding gastown after merge..."
go install ./...
echo "✓ Installed to ~/go/bin"
