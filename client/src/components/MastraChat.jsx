import React, { useState } from "react";
import axios from "axios";

const MASRTA_API_URL = "http://localhost:4111/api/agents/songAgent/generate";

const MastraChat = () => {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);

  const sendMessage = async () => {
    if (!input.trim()) return;

    const userMsg = { role: "user", content: input };
    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setLoading(true);

    try {
      const response = await axios.post(MASRTA_API_URL, {
        messages: [{ role: "user", content: input }],
      });

      console.log("✅ Mastra agent raw response:", response.data);

      // 🎯 Directly extract the text key (as shown by your curl output)
      const reply =
        response.data?.text ||
        response.data?.output?.text ||
        "⚠️ No response text found from agent.";

      setMessages((prev) => [...prev, { role: "agent", content: reply }]);
    } catch (err) {
      console.error("Mastra chat error:", err.response?.data || err.message);
      setMessages((prev) => [
        ...prev,
        { role: "agent", content: "⚠️ Failed to connect to Mastra Agent API." },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mastra-chat bg-white shadow-lg rounded-xl p-4 mt-6 w-full max-w-xl mx-auto border border-gray-200">
      <h3 className="text-xl font-semibold text-gray-800 mb-3 text-center">
        💬 Chat with your Song Agent
      </h3>

      <div
        className="chat-window h-64 overflow-y-auto border rounded-md p-3 mb-3"
        style={{ backgroundColor: "#f9fafb" }}
      >
        {messages.map((msg, i) => (
          <div
            key={i}
            className={`mb-2 ${
              msg.role === "user"
                ? "text-right text-blue-700"
                : "text-left text-gray-700"
            }`}
          >
            <span
              className={`inline-block px-3 py-2 rounded-lg ${
                msg.role === "user"
                  ? "bg-blue-100 text-blue-800"
                  : "bg-gray-200 text-gray-800"
              }`}
              dangerouslySetInnerHTML={{ __html: msg.content }}
            ></span>
          </div>
        ))}
        {loading && <p className="text-gray-400 italic">Agent is thinking...</p>}
      </div>

      <div className="flex">
        <input
          type="text"
          placeholder="Ask about any song..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && sendMessage()}
          className="flex-1 border rounded-l-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
        />
        <button
          onClick={sendMessage}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-r-md"
        >
          Send
        </button>
      </div>
    </div>
  );
};

export default MastraChat;