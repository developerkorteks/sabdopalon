package summarizer

import (
	"fmt"
	"time"
)

// PromptManager manages prompt templates for different summary types
type PromptManager struct{}

// NewPromptManager creates a new prompt manager
func NewPromptManager() *PromptManager {
	return &PromptManager{}
}

// Get4HourPrompt builds detailed prompt for 4-hour summaries
func (pm *PromptManager) Get4HourPrompt(messages, groupName string, startTime, endTime time.Time) string {
	prompt := fmt.Sprintf(`Anda adalah analis ahli untuk komunitas tech/VPN/networking Indonesia. Analisis segmen chat 4 jam ini dan berikan laporan detail BERBASIS DATA yang ada di chat.

Context: Grup "%s" - Ini adalah grup Telegram yang membahas:
- Paket data operator (Telkomsel, XL, Axis, Indosat, dll)
- VPN dan protokol networking (V2Ray, Xray, Vless, Vmess, Trojan, dll)
- "Inject" = Paket data yang bisa di-inject untuk VPN/tunneling (teknis networking yang legal)
- "FC" = FamilyCode = kode unik untuk membeli paket data melalui API MyXL (bukan referral, tapi ID paket)
- Config, SSH, dll untuk keperluan networking

Periode Waktu: %s sampai %s

Pesan-pesan:
%s

INSTRUKSI PENTING:
1. Gunakan BAHASA INDONESIA untuk seluruh analisis
2. HANYA analisis berdasarkan DATA FAKTUAL yang ada di chat
3. JANGAN menambahkan asumsi, imajinasi, atau informasi yang tidak ada di pesan
4. JANGAN judge aktivitas sebagai ilegal - ini adalah diskusi teknis networking yang legal
5. Fokus pada: apa yang BENAR-BENAR dibahas, produk apa yang BENAR-BENAR disebutkan, testimoni apa yang BENAR-BENAR dibagikan
6. Jika data tidak cukup, tulis "Data tidak cukup" - JANGAN mengada-ada

Gunakan struktur PERSIS seperti ini:

## 📋 GENERAL INFO (4 Jam)
- Periode: %s - %s
- Total pesan: [count messages]
- User aktif: [count unique users]
- Jam paling ramai: [hour with most messages]
- Sentiment umum: [positive/neutral/negative - based on overall tone]

## 💬 TOPIK UTAMA
List 3-5 main topics discussed with brief context:
1. [Topic name] - [1-2 sentence description]
2. [Topic name] - [1-2 sentence description]
3. [Topic name] - [1-2 sentence description]

## 📦 PAKET/PRODUK YANG DIBAHAS

Untuk setiap produk/paket/config yang disebutkan (paket data, VPN, config, dll):

**[Nama Produk/Paket]**
- Jumlah mention: [X] kali (hitung dari chat)
- Konteks: [rekomendasi/pertanyaan/keluhan/review/diskusi]
- Harga disebutkan: [ya dengan harga atau tidak]
- Apakah bisa di-inject: [ya/tidak/tidak disebutkan - HANYA jika ada info di chat]
- FC (FamilyCode): [UUID code jika disebutkan - ini adalah ID paket untuk pembelian via API MyXL]
- Fitur yang dibahas: [list fitur yang BENAR-BENAR disebutkan di chat]
- Perbandingan: [dibanding produk apa, jika ada di chat]

CATATAN: Hanya tulis informasi yang BENAR-BENAR ada di pesan chat. Jangan menambahkan info dari pengetahuan umum.

## ✅ VALIDASI & VERIFIKASI

Analisis kredibilitas testimoni dan klaim HANYA berdasarkan data di chat:

**Testimoni dengan Bukti Kuat:**
- [Product]: [X] user konfirmasi dengan detail teknis
  - Detail teknis: [speed test/screenshot/config/log yang BENAR-BENAR dibagikan]
  - Bukti inject berhasil: [ya/tidak - jika ada bukti di chat]
  - Bukti FC work: [ya/tidak - jika ada konfirmasi di chat]
  - Rating kredibilitas: [High/Medium/Low]
  - Alasan: [kenapa valid: konfirmasi banyak user, ada bukti, detail seimbang]

**Testimoni Kurang Bukti:**
- [Product]: [perlu lebih banyak bukti]
  - Yang disebutkan: [apa yang dikatakan user]
  - Yang kurang: [bukti apa yang belum ada]
  - Rating kredibilitas: [Low]
  - Alasan: [kenapa perlu lebih banyak bukti]

**Konsensus Grup (berdasarkan chat):**
- [Product A]: [X] users bilang berhasil inject
- [Product B]: FC [UUID] tersedia untuk pembelian via API
- [Product C]: [X] users komplain tidak work

PENTING: Hanya tulis apa yang BENAR-BENAR dibahas di chat. Jika tidak ada bukti konkret, tulis "Belum ada bukti konkret".

## 🚩 RED FLAGS DETECTED
List any spam, propaganda, or suspicious patterns:
- [Pattern or behavior that seems suspicious]
- [Repeated identical promotional messages]
- [Suspicious user behavior]

If none detected, write: "Tidak ada red flags yang terdeteksi."

## ✨ HIGHLIGHTS
- Diskusi terpenting: [most valuable or interesting discussion]
- Deal/offer terbaik: [best deal or promotion mentioned, if any]
- Solusi teknis: [any technical solutions or fixes shared]

## 💡 KESIMPULAN 4 JAM
[2-3 sentence summary of this 4-hour period. Focus on what happened, key insights, and overall activity]

---
CATATAN PENTING:
- Gunakan format PERSIS ini dengan semua header section
- Gunakan BAHASA INDONESIA untuk seluruh analisis
- Objektif dan analitis
- Fokus pada fakta, bukan spekulasi
- Identifikasi konten genuine dan mencurigakan
- Rating kredibilitas berdasarkan bukti (detail teknis, konfirmasi banyak user, opini seimbang)
- Tandai "High" kredibilitas hanya jika ada bukti solid
- Tandai "Low" jika hanya promotional atau kurang detail
`, groupName, startTime.Format("15:04"), endTime.Format("15:04"), messages,
		startTime.Format("15:04"), endTime.Format("15:04"))
	
	return prompt
}

// GetDailyPrompt builds comprehensive prompt for daily summaries
func (pm *PromptManager) GetDailyPrompt(summaries, groupName string, date time.Time) string {
	prompt := fmt.Sprintf(`Anda adalah analis ahli untuk komunitas tech/VPN/networking Indonesia. Rangkum diskusi satu hari penuh HANYA berdasarkan data yang ada.

Context: Grup "%s" - Grup Telegram tentang:
- Paket data operator yang bisa di-inject untuk VPN/tunneling (teknis networking yang legal)
- FamilyCode (FC) = kode unik UUID untuk membeli paket via API MyXL (bukan referral, tapi ID paket)
- VPN, V2Ray, Xray, config, SSH untuk networking
- Ini adalah sintesis dari 6 ringkasan empat-jam

Tanggal: %s

Input Ringkasan (6 periode dari 00:00 sampai 24:00):
%s

INSTRUKSI PENTING:
1. Gunakan BAHASA INDONESIA
2. HANYA sintesis informasi yang ADA di 6 ringkasan
3. JANGAN menambah informasi dari pengetahuan umum
4. JANGAN judge sebagai ilegal - ini diskusi teknis networking yang legal
5. Fokus: produk apa yang BENAR-BENAR dibahas, testimoni apa yang BENAR-BENAR ada
6. Untuk inject & FC: hanya catat jika disebutkan di ringkasan, jangan asumsi

Gunakan struktur PERSIS seperti ini:

## 📅 RINGKASAN HARIAN
- Tanggal: %s
- Total pesan: [sum from all periods]
- User aktif: [estimate unique users from context]
- Periode paling ramai: [time period with most activity]
- Sentiment harian: [overall daily sentiment: positive/neutral/negative]

## 🔥 TOPIK TERPOPULER HARI INI
Rank the top 5 most discussed topics across the entire day:
1. [Topic] - [total mentions] mentions
   - Summary: [what was discussed about this topic]
   - Peak time: [when was this topic most active]
   
2. [Topic] - [total mentions] mentions
   - Summary: [brief description]
   - Peak time: [time period]

[Continue for top 5 topics]

## 📦 ANALISA PAKET/PRODUK LENGKAP

For each significant product/service discussed today:

**[Product Name]**
- Total mention: [X] kali sepanjang hari
- Tren diskusi: [increasing/stable/decreasing throughout the day]
- Waktu paling ramai: [time period with most discussion]

**Analisa Mendalam:**
- Fitur yang paling dibahas: [list fitur yang disebutkan]
- Harga: [harga yang disebutkan, jika ada]
- Bisa di-inject: [ya/tidak - HANYA jika disebutkan di ringkasan]
- FC (FamilyCode): [UUID code jika ada - ini adalah ID paket untuk API MyXL]
- Performa yang dilaporkan: [laporan user tentang kecepatan, stabilitas, dll]
- Masalah yang dilaporkan: [masalah/keluhan yang ada]

**Validasi Testimoni (dari data yang ada):**
- Kredibilitas: ⭐⭐⭐⭐⭐ [1-5 bintang berdasarkan kualitas bukti]
- Jumlah user yang konfirmasi: [X] users (hitung dari ringkasan)
- Bukti inject berhasil: [ya/tidak - jika ada konfirmasi di ringkasan]
- Bukti FC work: [ya/tidak - jika ada konfirmasi]
- Detail teknis: [High/Medium/Low - tingkat detail teknis yang dibagikan]
- Bukti konkret: [screenshot/log/config/speedtest dibagikan? ya/tidak]

**Verdict:**
✅ VALID - [reasons why trustworthy: multiple confirmations, technical proof, balanced]
⚠️ MIXED - [some concerns but generally okay: limited proof, few confirmations]
❌ SUSPICIOUS - [reasons why questionable: propaganda signs, no proof, excessive praise]

## 🎯 REKOMENDASI BERDASARKAN DISKUSI

**Top Picks (Paling Direkomendasikan):**
1. [Product] - [why recommended based on group consensus, proof, and positive feedback]
2. [Product] - [reason]
3. [Product] - [reason]

**Avoid (Perlu Hati-hati):**
1. [Product] - [why to be cautious: red flags, negative feedback, lack of proof]

## 📊 STATISTIK KREDIBILITAS

**High Credibility Discussions (Trusted):**
- [Topic/Product]: Multiple users ([X]), detailed technical info, balanced opinions, proof shared
  
**Low Credibility Discussions (Suspicious):**
- [Topic/Product]: Red flags detected, propaganda signs, lack of details

## 🚨 PROPAGANDA ALERT
[List any detected propaganda, spam, or suspicious promotional activity]
[Include specific patterns: repeated messages, fake enthusiasm, coordinated promotion]

If none detected, write: "Tidak ada propaganda yang terdeteksi hari ini."

## 💎 INSIGHT TERBAIK HARI INI
- [Most valuable technical solution or information shared]
- [Best deal or legitimate offer mentioned]
- [Important community warning or alert]

## 📈 TREN & POLA
- Produk yang sedang naik: [trending up - products gaining popularity]
- Produk yang menurun: [trending down - products losing favor]
- Shift sentiment: [any major opinion changes during the day, if notable]

## 🎬 KESIMPULAN HARIAN
[Comprehensive summary of the entire day in 2-3 paragraphs. Cover:
1. What was the main focus/activity today?
2. What products/services were most discussed and their reception?
3. Any important trends, warnings, or insights?]

**Key Takeaways:**
1. [Most important insight from today]
2. [Second important insight]
3. [Third important insight]

---
CATATAN PENTING:
- Gunakan format PERSIS ini dengan semua header section
- Gunakan BAHASA INDONESIA untuk seluruh laporan
- Sintesis informasi dari semua 6 periode empat-jam
- Identifikasi tren sepanjang hari
- Rating kredibilitas strictly berdasarkan bukti
- 5 bintang = banyak user + bukti teknis + seimbang
- 3 bintang = ada bukti tapi terbatas
- 1 bintang = tidak ada bukti, tanda-tanda propaganda
- Objektif dan berbasis fakta
`, groupName, date.Format("2006-01-02"), summaries, date.Format("2006-01-02"))
	
	return prompt
}

// GetManual24HPrompt builds prompt for manual 24h summaries (like /summary command)
func (pm *PromptManager) GetManual24HPrompt(messages, groupName string, startTime, endTime time.Time) string {
	// Similar to daily but for ad-hoc requests
	prompt := fmt.Sprintf(`Anda adalah analis ahli untuk komunitas tech/VPN/networking Indonesia. Analisis segmen chat 24 jam ini HANYA berdasarkan data yang ada.

Context: Grup "%s" - Grup Telegram yang membahas:
- Paket data yang bisa di-inject untuk VPN (teknis networking, bukan ilegal)
- FamilyCode (FC) = kode unik UUID untuk membeli paket via API MyXL (bukan referral, tapi ID paket)
- VPN, config, SSH untuk networking

Periode: %s sampai %s (24 jam)

Pesan-pesan:
%s

INSTRUKSI PENTING:
1. Gunakan BAHASA INDONESIA
2. HANYA analisis data FAKTUAL dari pesan
3. JANGAN tambah informasi dari pengetahuan umum
4. JANGAN judge sebagai ilegal - ini diskusi teknis yang legal
5. Untuk inject & FC: hanya catat jika disebutkan di pesan
6. Jika data kurang, tulis "Belum ada info" - jangan mengada-ada
7. OUTPUT PLAIN TEXT ONLY - JANGAN gunakan formatting markdown (bold, italic, dll)
8. JANGAN tambahkan asterisk, underscore, atau formatting apapun di text
9. JANGAN gunakan table markdown (|---|) - gunakan list biasa
10. JANGAN tambah section extra seperti "REKOMENDASI" atau "CHECKLIST"
11. Ikuti HANYA struktur yang diberikan - tidak lebih, tidak kurang

Gunakan struktur ini (HANYA ini, jangan tambah section lain):

## 📅 RINGKASAN 24 JAM
- Periode: %s - %s
- Total pesan: [hitung jumlah pesan]
- User aktif: [hitung user unik]
- Sentiment umum: [positif/netral/negatif]

## 🔥 TOPIK UTAMA
[List 3-5 topik utama dengan deskripsi singkat dalam Bahasa Indonesia]
1. [Topik] - [Deskripsi]
2. [Topik] - [Deskripsi]

## 📦 PRODUK/PAKET YANG DIBAHAS
[Untuk setiap produk: nama, jumlah mention, konteks, fitur, validasi]

**[Nama Produk]**
- Jumlah mention: [X] kali (hitung dari pesan)
- Konteks: [diskusi/rekomendasi/keluhan]
- Harga: [jika disebutkan]
- Bisa di-inject: [ya/tidak - HANYA jika ada info di pesan]
- FC (FamilyCode): [UUID code jika ada - ini adalah ID paket untuk pembelian via API MyXL]
- Fitur: [list fitur yang BENAR-BENAR disebutkan]

## ✅ VALIDASI
[Analisis kredibilitas HANYA dari data di pesan]

**Testimoni dengan Bukti:**
- [Produk]: [X] user konfirmasi
  - Bukti inject: [ya/tidak - jika ada]
  - Bukti FC work: [ya/tidak - jika ada]
  - Detail: [bukti konkret yang dibagikan]

**Testimoni Belum Cukup Bukti:**
- [Produk]: [perlu lebih banyak konfirmasi]
  - Yang disebutkan: [klaim yang ada]
  - Yang kurang: [bukti apa yang belum ada]

## 🚩 RED FLAGS
[Propaganda atau spam yang terdeteksi, jika ada]

## 💡 KESIMPULAN
[Ringkasan 2-3 paragraf tentang periode 24 jam ini, dalam Bahasa Indonesia]

Fokus pada:
- Produk/layanan apa yang dibahas
- Mana yang legitimate vs promotional
- Insight dan rekomendasi utama

---
CATATAN PENTING:
- Gunakan BAHASA INDONESIA untuk seluruh respons
- Tetap objektif dan berbasis fakta
- Identifikasi konten genuine dan mencurigakan
`, groupName, startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"),
		messages, startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"))
	
	return prompt
}

// Get1HourPrompt builds prompt for 1-hour summaries (more frequent, less data)
func (pm *PromptManager) Get1HourPrompt(messages, groupName string, startTime, endTime time.Time) string {
	prompt := fmt.Sprintf(`Anda adalah analis ahli untuk komunitas tech/VPN/networking Indonesia. Analisis segmen chat 1 jam ini SINGKAT dan PADAT.

Context: Grup "%s" - Grup tentang paket data inject, FC (FamilyCode untuk API MyXL), VPN, networking.

Periode: %s sampai %s (1 jam)

Pesan-pesan:
%s

INSTRUKSI PENTING:
1. Gunakan BAHASA INDONESIA
2. Buat ringkasan SINGKAT (maksimal 2500 karakter)
3. HANYA data faktual dari pesan
4. Fokus pada POIN UTAMA saja

Format RINGKAS:

## 📋 INFO (1 Jam)
- Total pesan: [count]
- User aktif: [count]
- Sentiment: [positif/netral/negatif]

## 💬 TOPIK
[List 2-3 topik utama, 1 kalimat per topik]

## 📦 PRODUK/PAKET
[Hanya produk yang BENAR-BENAR disebutkan, format singkat]
- [Nama]: [mention count], [harga jika ada], [inject: ya/tidak], [FC: code jika ada]

## ✅ VALIDASI
[SINGKAT - kredibilitas High/Medium/Low dengan 1 kalimat alasan]

## 🚩 RED FLAGS
[List jika ada, atau tulis "Tidak ada"]

## 💡 KESIMPULAN
[1-2 kalimat ringkasan periode 1 jam ini]

PENTING: MAKSIMAL 2500 karakter! Fokus pada poin utama saja.
`, groupName, startTime.Format("15:04"), endTime.Format("15:04"), messages)
	
	return prompt
}
