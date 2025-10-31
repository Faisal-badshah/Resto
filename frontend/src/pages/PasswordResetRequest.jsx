import React, { useState } from "react";
import { requestPasswordReset } from "../api";

export default function PasswordResetRequest() {
  const [restaurantId, setRestaurantId] = useState(process.env.REACT_APP_DEFAULT_RESTAURANT || "1");
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState("");

  const submit = async () => {
    if (!email) return setStatus("Enter your email");
    setStatus("Sending reset email...");
    try {
      await requestPasswordReset(parseInt(restaurantId, 10), email);
      setStatus("If that email exists, a reset link has been sent.");
    } catch (e) {
      console.error(e);
      setStatus("Failed to request reset (try again later).");
    }
  };

  return (
    <div style={{ padding: 20 }}>
      <h2>Reset admin password</h2>
      <div>
        <label>Restaurant ID</label><br/>
        <input value={restaurantId} onChange={e => setRestaurantId(e.target.value)} />
      </div>
      <div>
        <label>Admin Email</label><br/>
        <input value={email} onChange={e => setEmail(e.target.value)} />
      </div>
      <div style={{ marginTop: 12 }}>
        <button onClick={submit}>Send password reset link</button>
      </div>
      <div style={{ marginTop: 12 }}>{status}</div>
    </div>
  );
}
