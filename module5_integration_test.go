package main

import (
	"strings"
	"testing"
	"time"

	"money/internal/categorization"
	"money/internal/statement"
)

func TestModule5RuleClassificationAndCorrectionLearning(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(ctx, "Module Five Integration User")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		deleteTestUser(t, store, user.ID)
	})

	categories, err := store.listCategories(ctx, user.ID)
	if err != nil {
		t.Fatalf("list seeded categories: %v", err)
	}
	if len(categories) != len(defaultCategoryNames) {
		t.Fatalf("seeded category count = %d, want %d", len(categories), len(defaultCategoryNames))
	}

	groceries, err := findCategoryByName(ctx, store.db, user.ID, "Groceries")
	if err != nil {
		t.Fatalf("find Groceries category: %v", err)
	}
	restaurants, err := findCategoryByName(ctx, store.db, user.ID, "Restaurants")
	if err != nil {
		t.Fatalf("find Restaurants category: %v", err)
	}

	createdRule, err := store.createMerchantRule(ctx, NewMerchantRule{
		UserID:       user.ID,
		Source:       statement.Swedbank,
		MatchType:    categorization.MatchExact,
		Pattern:      "MAXIMA",
		MerchantName: "Maxima",
		CategoryName: groceries.Name,
		Priority:     100,
	})
	if err != nil {
		t.Fatalf("create merchant rule: %v", err)
	}

	firstInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT-MODULE-5,2026-08-28,MAXIMA,Purchase,25.50,EUR,D,module-5-record-1,CARD\n"

	firstImport, err := importStatement(
		ctx,
		store,
		user.ID,
		"module-5-first.csv",
		strings.NewReader(firstInput),
		time.UTC,
	)
	if err != nil {
		t.Fatalf("import first statement: %v", err)
	}
	t.Cleanup(func() {
		for _, transaction := range firstImport.Stored.Transactions {
			deleteTestTransaction(t, store, transaction.ID)
		}
		deleteTestStatement(t, store, firstImport.Stored.Statement.ID)
	})

	if firstImport.Stored.RuleClassified != 1 {
		t.Errorf("first import rule-classified count = %d, want 1", firstImport.Stored.RuleClassified)
	}
	if len(firstImport.Stored.CategorizationConflicts) != 0 {
		t.Errorf("first import conflicts = %+v, want none", firstImport.Stored.CategorizationConflicts)
	}

	firstTransactionID := firstImport.Stored.Transactions[0].ID
	listed, err := store.listTransactionsByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("list classified transactions: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed transaction count = %d, want 1", len(listed))
	}
	if listed[0].MerchantName != "Maxima" ||
		listed[0].CategoryName != "Groceries" ||
		listed[0].Classification != "rule" {
		t.Errorf("first classification = merchant %q, category %q, source %q", listed[0].MerchantName, listed[0].CategoryName, listed[0].Classification)
	}

	merchant, _, err := store.applyManualCorrection(ctx, user.ID, ManualCorrection{
		TransactionID: firstTransactionID,
		CategoryID:    restaurants.ID,
		MerchantName:  "Maxima",
	})
	if err != nil {
		t.Fatalf("apply manual correction: %v", err)
	}

	learnedRule, err := store.learnMerchantRule(ctx, NewMerchantRule{
		UserID:       user.ID,
		Source:       statement.Swedbank,
		MatchType:    categorization.MatchExact,
		Pattern:      "MAXIMA",
		MerchantName: merchant.Name,
		CategoryName: restaurants.Name,
		Priority:     100,
	})
	if err != nil {
		t.Fatalf("learn corrected merchant rule: %v", err)
	}
	if learnedRule.ID != createdRule.ID {
		t.Errorf("learned rule ID = %d, want existing rule ID %d", learnedRule.ID, createdRule.ID)
	}

	secondInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT-MODULE-5,2026-08-29,MAXIMA,Another purchase,31.00,EUR,D,module-5-record-2,CARD\n"

	secondImport, err := importStatement(
		ctx,
		store,
		user.ID,
		"module-5-second.csv",
		strings.NewReader(secondInput),
		time.UTC,
	)
	if err != nil {
		t.Fatalf("import second statement: %v", err)
	}
	t.Cleanup(func() {
		for _, transaction := range secondImport.Stored.Transactions {
			deleteTestTransaction(t, store, transaction.ID)
		}
		deleteTestStatement(t, store, secondImport.Stored.Statement.ID)
	})

	if secondImport.Stored.RuleClassified != 1 {
		t.Errorf("second import rule-classified count = %d, want 1", secondImport.Stored.RuleClassified)
	}

	var secondCategory string
	err = store.db.QueryRowContext(
		ctx,
		`SELECT c.name
		 FROM transactions AS t
		 JOIN categories AS c ON c.id = t.category_id
		 WHERE t.id = $1`,
		secondImport.Stored.Transactions[0].ID,
	).Scan(&secondCategory)
	if err != nil {
		t.Fatalf("query learned classification: %v", err)
	}
	if secondCategory != "Restaurants" {
		t.Errorf("learned category = %q, want Restaurants", secondCategory)
	}
}
