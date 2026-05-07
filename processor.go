package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	_ "github.com/go-sql-driver/mysql"
)

type Transaction struct {
	Date      string  `json:"date"`
	Merchant  string  `json:"merchant"`
	Category  string  `json:"category"`
	Location  string  `json:"location"`
	Amount    float64 `json:"amount"`
	Month     string  `json:"month"`
	IsCredit  bool    `json:"is_credit"`
}

type CategorySummary struct {
	Category     string
	Month        string
	Total        float64
	Count        int
	Avg          float64
	PctOfTotal   float64
}

type MonthlySummary struct {
	Month        string
	TotalSpend   float64
	TotalCredit  float64
	NetSpend     float64
	TxnCount     int
	AvgTxn       float64
	TopCategory  string
	TopMerchant  string
}

type MerchantSummary struct {
	Merchant   string
	Category   string
	Visits     int
	Total      float64
	AvgVisit   float64
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func main() {
	var txns []Transaction
	if err := json.NewDecoder(os.Stdin).Decode(&txns); err != nil {
		log.Fatalf("JSON decode error: %v", err)
	}

	db, err := sql.Open("mysql", "spending_user:spend123@tcp(127.0.0.1:3306)/spending_db")
	if err != nil {
		log.Fatalf("DB connect error: %v", err)
	}
	defer db.Close()

	// Clear old data
	for _, t := range []string{"transactions","category_summary","monthly_summary","merchant_summary"} {
		db.Exec("TRUNCATE TABLE " + t)
	}

	// Insert transactions
	stmt, _ := db.Prepare("INSERT INTO transactions (date,merchant,category,location,amount,month,is_credit) VALUES (?,?,?,?,?,?,?)")
	for _, t := range txns {
		stmt.Exec(t.Date, t.Merchant, t.Category, t.Location, t.Amount, t.Month, t.IsCredit)
	}
	stmt.Close()
	fmt.Printf("✓ Inserted %d transactions\n", len(txns))

	// ── Calculate category summaries ──
	catMonth := map[string]map[string]*CategorySummary{}
	monthTotal := map[string]float64{}

	for _, t := range txns {
		if t.IsCredit { continue }
		if catMonth[t.Month] == nil {
			catMonth[t.Month] = map[string]*CategorySummary{}
		}
		key := t.Category
		if catMonth[t.Month][key] == nil {
			catMonth[t.Month][key] = &CategorySummary{Category: t.Category, Month: t.Month}
		}
		catMonth[t.Month][key].Total = round2(catMonth[t.Month][key].Total + t.Amount)
		catMonth[t.Month][key].Count++
		monthTotal[t.Month] = round2(monthTotal[t.Month] + t.Amount)
	}

	stmt2, _ := db.Prepare("INSERT INTO category_summary (category,month,total_amount,transaction_count,avg_amount,pct_of_total) VALUES (?,?,?,?,?,?)")
	for month, cats := range catMonth {
		for _, cs := range cats {
			cs.Avg = round2(cs.Total / float64(cs.Count))
			cs.PctOfTotal = round2(cs.Total / monthTotal[month] * 100)
			stmt2.Exec(cs.Category, cs.Month, cs.Total, cs.Count, cs.Avg, cs.PctOfTotal)
		}
	}
	stmt2.Close()
	fmt.Println("✓ Calculated category summaries")

	// ── Calculate monthly summaries ──
	months := map[string]*MonthlySummary{}
	catTotalsAll := map[string]map[string]float64{}  // month -> cat -> total
	merchantVisits := map[string]map[string]int{}    // month -> merchant -> count

	for _, t := range txns {
		if months[t.Month] == nil {
			months[t.Month] = &MonthlySummary{Month: t.Month}
			catTotalsAll[t.Month] = map[string]float64{}
			merchantVisits[t.Month] = map[string]int{}
		}
		if t.IsCredit {
			months[t.Month].TotalCredit = round2(months[t.Month].TotalCredit + t.Amount)
		} else {
			months[t.Month].TotalSpend = round2(months[t.Month].TotalSpend + t.Amount)
			months[t.Month].TxnCount++
			catTotalsAll[t.Month][t.Category] = round2(catTotalsAll[t.Month][t.Category] + t.Amount)
			merchantVisits[t.Month][t.Merchant]++
		}
	}

	stmt3, _ := db.Prepare("INSERT INTO monthly_summary (month,total_spend,total_credit,net_spend,transaction_count,avg_transaction,top_category,top_merchant) VALUES (?,?,?,?,?,?,?,?)")
	for month, ms := range months {
		ms.NetSpend = round2(ms.TotalSpend - ms.TotalCredit)
		if ms.TxnCount > 0 {
			ms.AvgTxn = round2(ms.TotalSpend / float64(ms.TxnCount))
		}
		// top category
		topCatAmt := 0.0
		for cat, amt := range catTotalsAll[month] {
			if amt > topCatAmt { topCatAmt = amt; ms.TopCategory = cat }
		}
		// top merchant
		topCount := 0
		for merch, cnt := range merchantVisits[month] {
			if cnt > topCount { topCount = cnt; ms.TopMerchant = merch }
		}
		stmt3.Exec(ms.Month, ms.TotalSpend, ms.TotalCredit, ms.NetSpend, ms.TxnCount, ms.AvgTxn, ms.TopCategory, ms.TopMerchant)
	}
	stmt3.Close()
	fmt.Println("✓ Calculated monthly summaries")

	// ── Calculate merchant summaries ──
	merchants := map[string]*MerchantSummary{}
	for _, t := range txns {
		if t.IsCredit { continue }
		if merchants[t.Merchant] == nil {
			merchants[t.Merchant] = &MerchantSummary{Merchant: t.Merchant, Category: t.Category}
		}
		merchants[t.Merchant].Visits++
		merchants[t.Merchant].Total = round2(merchants[t.Merchant].Total + t.Amount)
	}

	stmt4, _ := db.Prepare("INSERT INTO merchant_summary (merchant,category,visit_count,total_spent,avg_per_visit) VALUES (?,?,?,?,?)")
	for _, ms := range merchants {
		ms.AvgVisit = round2(ms.Total / float64(ms.Visits))
		stmt4.Exec(ms.Merchant, ms.Category, ms.Visits, ms.Total, ms.AvgVisit)
	}
	stmt4.Close()
	fmt.Println("✓ Calculated merchant summaries")

	// ── Print summary report ──
	fmt.Println("\n═══════════════════════════════")
	fmt.Println("  SPENDING SYSTEM — RESULTS")
	fmt.Println("═══════════════════════════════")

	monthList := []string{}
	for m := range months { monthList = append(monthList, m) }
	sort.Strings(monthList)

	for _, m := range monthList {
		ms := months[m]
		fmt.Printf("\n📅 %s\n", m)
		fmt.Printf("   Total Spend:   $%.2f\n", ms.TotalSpend)
		fmt.Printf("   Total Credit:  $%.2f\n", ms.TotalCredit)
		fmt.Printf("   Net Spend:     $%.2f\n", ms.NetSpend)
		fmt.Printf("   Transactions:  %d\n", ms.TxnCount)
		fmt.Printf("   Avg/Txn:       $%.2f\n", ms.AvgTxn)
		fmt.Printf("   Top Category:  %s\n", ms.TopCategory)
		fmt.Printf("   Top Merchant:  %s\n", ms.TopMerchant)
	}

	fmt.Println("\n✅ All data written to MySQL spending_db")
}
