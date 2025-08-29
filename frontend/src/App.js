import React, { useState, memo } from "react";
import styled from "styled-components";


const Container = styled.div`
    font-family: sans-serif;
    max-width: 800px;
    margin: auto;
    padding: 20px;

    h1, h2 {
        border-bottom: 2px solid #eee;
        padding-bottom: 10px;
    }

    button {
        padding: 10px 15px;
        font-size: 16px;
        cursor: pointer;
        margin-right: 10px;
    }

    pre {
        background-color: #f4f4f4;
        padding: 15px;
        border-radius: 5px;
        white-space: pre-wrap;
        word-wrap: break-word;
    }

    .step {
        margin-bottom: 30px;
    }

    .status {
        font-weight: bold;
    }
    `;

const App =  memo (() => {
    const [signedMessage, setSignedMessage] = useState(null);
    const [getResponse, setGetResponse] = useState("(Not called yet)");
    const [submitResponse, setSubmitResponse] = useState("(Waiting for Step 1)");
    const [viewResponse, setViewResponse] = useState("(Not called yet)");
    const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);

    // Step 1: /sign-test 호출
    const handleGetResponse = async () => {
        setGetResponse("Loading...");
        try {
            const res = await fetch("http://localhost:8080/sign-test");
            if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
            const data = await res.json();
            setSignedMessage(data);  // 받은 데이터 저장
            setGetResponse(JSON.stringify(data, null, 2));
            setSubmitResponse("(Ready to submit)");
            setIsSubmitDisabled(false);  // 2단계 버튼 활성화
        } catch (err) {
            setGetResponse(`Error: ${err.message}`);
        }
};

// Step 2: /submit 호출
const handleSubmitResponse = async () => {
    if (!signedMessage) {
        setSubmitResponse("Error: No payload from Step 1.");
      return;
    }

    setSubmitResponse("Loading...");
    try {
        const res = await fetch("http://localhost:8080/submit", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(signedMessage),
        });
        const text = await res.text();
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status} - ${text}`);
        setSubmitResponse(`Success: ${text}`);
    } catch (err) {
        setSubmitResponse(`Error: ${err.message}`);
    }
};

// Step 3: /messages 호출
const handleViewResponse = async () => {
    setViewResponse("Loading...");
    try {
        const res = await fetch("http://localhost:8080/messages");
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
        const data = await res.json();
        setViewResponse(JSON.stringify(data, null, 2));
    } catch (err) {
        setViewResponse(`Error: ${err.message}`);
    }
};

return (
    <Container>
        <h1>Go Web3 Server Frontend Test</h1>

        <div className="step">
            <h2>Step 1: Get Signed Message Sample</h2>
            <button onClick={handleGetResponse}>/sign-test 호출</button>
            <p className="status">Response:</p>
            <pre>{getResponse}</pre>
        </div>

        <div className="step">
            <h2>Step 2: Submit for Verification</h2>
            <button onClick={handleSubmitResponse} disabled={isSubmitDisabled}>/submit 호출</button>
            <p className="status">Response:</p>
            <pre>{submitResponse}</pre>
        </div>

        <div className="step">
            <h2>Step 3: View All Verified Messages</h2>
            <button onClick={handleViewResponse}>/messages 호출</button>
            <p className="status">Response:</p>
            <pre>{viewResponse}</pre>
        </div>
    </Container>
  );
});

export default App;