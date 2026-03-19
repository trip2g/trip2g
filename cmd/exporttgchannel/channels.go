package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"trip2g/internal/tgtd"
)

func runChannels(ctx context.Context, sessionPath string, args []string) error {
	fs := flag.NewFlagSet("channels", flag.ExitOnError)
	filter := fs.String("filter", "", "filter by title substring (case-insensitive)")
	fs.Usage = func() {
		fmt.Println("usage: exporttgchannel channels [--filter NAME]\n\nLists channels the account has access to, with their numeric IDs.\nUse the ID with 'export --channel ID'.")
	}
	_ = fs.Parse(args)

	sess, err := loadSession(sessionPath)
	if err != nil {
		return err
	}
	sessionData, err := sess.sessionBytes()
	if err != nil {
		return err
	}

	client := tgtd.NewClient(&cliEnv{}, 0, sess.APIID, sess.APIHash)

	fmt.Println("Fetching dialogs...")
	dialogs, err := client.ListDialogs(ctx, sessionData, 0)
	if err != nil {
		return fmt.Errorf("list dialogs: %w", err)
	}

	needle := strings.ToLower(*filter)
	found := 0
	fmt.Printf("%-20s  %s\n", "ID", "Title")
	fmt.Println(strings.Repeat("-", 60))
	for _, d := range dialogs {
		if d.Type != tgtd.DialogTypeChannel {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(d.Title), needle) {
			continue
		}
		username := ""
		if d.Username != "" {
			username = " (@" + d.Username + ")"
		}
		fmt.Printf("%-20d  %s%s\n", d.ID, d.Title, username)
		found++
	}
	if found == 0 {
		fmt.Println("No channels found.")
	}
	return nil
}
