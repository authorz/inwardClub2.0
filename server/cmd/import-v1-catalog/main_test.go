package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDumpAndMappings(t *testing.T) {
	dump := "INSERT INTO `categories` (`id`,`store_id`,`parent_id`,`level`,`name`,`image`,`sort_order`,`status`,`created_at`,`updated_at`) VALUES (1,1,NULL,1,'饮品','uploads/c.jpg',2,'active','2025-01-01 00:00:00','2025-01-02 00:00:00');\n" +
		"INSERT INTO `products` (`id`,`store_id`,`category_id`,`name`,`image`,`description`,`price`,`stock`,`type`,`points`,`sort_order`,`status`,`created_at`,`updated_at`) VALUES (7,1,1,'老板\\'特调','uploads/p.jpg','柠檬, 苏打',42.50,9,1,2100,3,'active','2025-01-01 00:00:00','2025-01-02 00:00:00');\n"
	filename := filepath.Join(t.TempDir(), "dump.sql")
	if err := os.WriteFile(filename, []byte(dump), 0o600); err != nil {
		t.Fatal(err)
	}
	categories, products, err := parseDump(filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 1 || categories[0].Name != "饮品" || categories[0].ParentID != nil {
		t.Fatalf("unexpected categories: %+v", categories)
	}
	if len(products) != 1 || products[0].Name != "老板'特调" || products[0].Description != "柠檬, 苏打" {
		t.Fatalf("unexpected products: %+v", products)
	}
	if products[0].PriceCent != 4250 || products[0].Points != 2100 {
		t.Fatalf("unexpected money/points mapping: %+v", products[0])
	}
	if err := validateLegacyData(categories, products); err != nil {
		t.Fatal(err)
	}
}

func TestDecimalToCent(t *testing.T) {
	for input, want := range map[string]int64{"0": 0, "1.2": 120, "68.00": 6800, "168.99": 16899} {
		got, err := decimalToCent(input)
		if err != nil || got != want {
			t.Fatalf("decimalToCent(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
}
