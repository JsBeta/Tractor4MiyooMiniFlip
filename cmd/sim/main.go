package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"tractor/internal/game"
)

func main() {
	n := flag.Int("n", 50, "rounds")
	flag.Parse()
	var scores []int
	var failures int
	start := time.Now()
	for seed := int64(1); seed <= int64(*n); seed++ {
		g := game.NewGame(seed)
		for round := 0; round < 4 && g.Winner < 0; round++ {
			g.StartRound()
			r := g.Current
			for r.Phase == game.PhaseBidding {
				seat := r.BidSeat()
				if cards, ok := game.AIBid(r, seat); ok {
					if err := r.Bid(seat, cards); err != nil {
						fmt.Println("bid error", seed, err)
						failures++
						break
					}
				} else if err := r.PassBid(seat); err != nil {
					fmt.Println("pass error", seed, err)
					failures++
					break
				}
			}
			if r.Phase == game.PhaseBury {
				if err := game.AIAutoBury(r); err != nil {
					fmt.Println("bury error", seed, err)
					failures++
				}
			}
			for r.Phase == game.PhasePlay {
				seat := r.Turn
				cards, err := game.AIChoose(r, seat)
				if err != nil {
					fmt.Println("choose error", seed, seat, err)
					failures++
					r.Phase = game.PhaseDone
					break
				}
				if err := r.Play(seat, cards); err != nil {
					fmt.Println("play error", seed, seat, err, cards)
					failures++
					r.Phase = game.PhaseDone
					break
				}
			}
			if r.Phase != game.PhaseDone {
				break
			}
			scores = append(scores, r.DefenderPoints)
			res := g.FinishRound()
			if res.Winner >= 0 {
				break
			}
		}
	}
	sum := 0
	hist := make(map[int]int)
	for _, s := range scores {
		sum += s
		hist[s/10*10]++
	}
	fmt.Printf("rounds=%d failures=%d avg=%.1f elapsed=%s\n", len(scores), failures, float64(sum)/math.Max(1, float64(len(scores))), time.Since(start))
	for k := 0; k <= 200; k += 10 {
		if hist[k] > 0 {
			fmt.Printf("  %3d-%3d: %d\n", k, k+9, hist[k])
		}
	}
	if failures > 0 {
		os.Exit(1)
	}
}
