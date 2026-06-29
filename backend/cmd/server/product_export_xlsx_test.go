package main

import "testing"

func TestProductPiecesTextUsesExistingPieces(t *testing.T) {
	got := productPiecesText(12, "银色1P\n黑色2P")
	if got != "12P" {
		t.Fatalf("productPiecesText() = %q, want %q", got, "12P")
	}
}

func TestProductPiecesTextCalculatesFromConfigWhenZero(t *testing.T) {
	config := "银色单角1P 金色凤翅1P 黑色凤翅1p\n白色仿真芦苇3P"
	got := productPiecesText(0, config)
	if got != "6P" {
		t.Fatalf("productPiecesText() = %q, want %q", got, "6P")
	}
}

func TestProductConfigPiecesIgnoresWordsLikePcs(t *testing.T) {
	config := "12pcs 暗夜白羽轻奢仿真花束\n银色1P 黑色2p"
	got := productConfigPieces(config)
	if got != 3 {
		t.Fatalf("productConfigPieces() = %d, want %d", got, 3)
	}
}

func TestProductConfigPiecesHandlesAdjacentChineseText(t *testing.T) {
	config := "白色柳叶1P黑色散尾2P金色羊角3p"
	got := productConfigPieces(config)
	if got != 6 {
		t.Fatalf("productConfigPieces() = %d, want %d", got, 6)
	}
}

func TestProductPiecesTextFallsBackToZero(t *testing.T) {
	got := productPiecesText(0, "未维护配置")
	if got != "0P" {
		t.Fatalf("productPiecesText() = %q, want %q", got, "0P")
	}
}
