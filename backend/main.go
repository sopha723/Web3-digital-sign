package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Message는 사용자가 서명할 데이터의 구조를 정의합니다.
type Message struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SignedMessage는 원본 메시지와 서명, 공개키를 함께 담습니다.
type SignedMessage struct {
	Message   Message `json:"message"`
	Signature string  `json:"signature"`
	PublicKey string  `json:"publicKey"`
}

// 검증된 메시지들을 저장하는 인메모리 슬라이스와 이를 보호하기 위한 뮤텍스입니다.
var (
	verifiedMessages []SignedMessage
	mu               sync.Mutex
)

// verify 함수는 공개키, 서명, 원본 데이터(바이트)를 받아 서명의 유효성을 검증합니다.
// 이 함수는 이제 올바른 키 역직렬화 로직을 포함합니다.
func verify(publicKeyHex string, signatureHex string, data []byte) bool {
	// 1. 16진수 문자열을 바이트 슬라이스로 디코딩합니다.
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false
	}
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}

	// 2. 바이트 슬라이스에서 ECDSA 공개키 객체를 복원합니다.
	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, publicKeyBytes)
	if x == nil {
		return false // Unmarshal 실패
	}
	publicKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	// 3. 원본 데이터의 해시를 계산합니다.
	hash := sha256.Sum256(data)

	// 4. 복원된 공개키를 사용하여 서명을 검증합니다.
	return ecdsa.VerifyASN1(publicKey, hash[:], sigBytes)
}

// handleSubmit는 클라이언트로부터 서명된 메시지를 받아 검증하고 저장하는 HTTP 핸들러입니다.
func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST 메서드만 허용됩니다.", http.StatusMethodNotAllowed)
		return
	}

	var signedMsg SignedMessage
	if err := json.NewDecoder(r.Body).Decode(&signedMsg); err != nil {
		http.Error(w, "잘못된 요청 본문입니다.", http.StatusBadRequest)
		return
	}

	msgBytes, err := json.Marshal(signedMsg.Message)
	if err != nil {
		http.Error(w, "메시지 직렬화 실패.", http.StatusInternalServerError)
		return
	}

	if !verify(signedMsg.PublicKey, signedMsg.Signature, msgBytes) {
		http.Error(w, "서명이 유효하지 않습니다.", http.StatusUnauthorized)
		return
	}

	// 뮤텍스를 사용하여 슬라이스에 대한 동시 접근을 제어합니다.
	mu.Lock()
	verifiedMessages = append(verifiedMessages, signedMsg)
	mu.Unlock()

	fmt.Printf("검증 성공 및 메시지 저장: %s\n", signedMsg.Message.Content)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "메시지가 성공적으로 검증 및 저장되었습니다.")
}

// handleGetMessages는 저장된 모든 검증된 메시지 목록을 JSON 형태로 반환합니다.
func handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET 메서드만 허용됩니다.", http.StatusMethodNotAllowed)
		return
	}

	mu.Lock() // 읽기 작업도 Lock을 거는 것이 안전합니다 (혹은 RWMutex 사용).
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	// CORS 접근 허용
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 슬라이스를 직접 인코딩하면 nil일 경우 'null'이 되므로, 빈 슬라이스를 만들어 처리합니다.
	if len(verifiedMessages) == 0 {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(verifiedMessages)
}

// handleSignTest는 키 생성, 메시지 생성, 서명 과정을 시뮬레이션하여 클라이언트에게 보여주는 학습용 핸들러입니다.
func handleSignTest(w http.ResponseWriter, r *http.Request) {
	// 1. 새 키 쌍 생성
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		http.Error(w, "키 생성 실패", http.StatusInternalServerError)
		return
	}

	// 2. 공개키를 표준 형식의 바이트 슬라이스로 변환 (Marshal 사용)
	publicKeyBytes := elliptic.Marshal(privateKey.PublicKey.Curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)
	publicKeyHex := hex.EncodeToString(publicKeyBytes)

	// 3. 테스트 메시지 생성
	msg := Message{
		Content:   "이것은 Go로 서명된 테스트 메시지입니다.",
		Timestamp: time.Now(),
	}
	msgBytes, _ := json.Marshal(msg)

	// 4. 메시지에 서명
	hash := sha256.Sum256(msgBytes)
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		http.Error(w, "서명 생성 실패", http.StatusInternalServerError)
		return
	}

	// 5. 서명된 메시지 객체 생성
	signedMsg := SignedMessage{
		Message:   msg,
		Signature: hex.EncodeToString(signature),
		PublicKey: publicKeyHex,
	}

	// 6. 결과를 JSON으로 클라이언트에게 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signedMsg)
}

// main 함수는 서버의 진입점입니다.
func main() {
	http.HandleFunc("/sign-test", handleSignTest)
	http.HandleFunc("/submit", handleSubmit)
	http.HandleFunc("/messages", handleGetMessages)

	fmt.Println("Web3 학습용 서버 시작 (포트: 8080)")
	fmt.Println("사용법:")
	fmt.Println("1. 브라우저나 curl로 'http://localhost:8080/sign-test'에 접속하여 서명된 메시지 샘플 확인")
	fmt.Println("2. 위 결과(JSON 전체)를 복사하여 '/submit' 엔드포인트에 POST 요청 전송")
	fmt.Println("3. 'http://localhost:8080/messages'에 접속하여 저장된 메시지 확인")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("서버 시작 실패: %v", err)
	}
}
