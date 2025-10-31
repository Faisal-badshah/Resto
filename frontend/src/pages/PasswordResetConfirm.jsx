import React, { useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { confirmPasswordReset } from "../api";

export default function PasswordResetConfirm() {
  const [params] = useSearchParams();
  const token = params.get("token");
  const restaurantId = params.get("restaurantId");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [status, setStatus] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    if (!token) setStatus("Invalid reset link");
  }, [token]);

  const submit = async () => {
    if (!token) return setStatus("Missing token");
    if (password.length < 8) return setStatus("Password must be at least 8 characters");
    if (password !== confirm) return setStatus("Passwords do not match");
    setStatus("Submitting...");
    try {
      await confirmPasswordReset(token, password);
      setStatus("Password set. Redirecting to admin login...");
      setTimeout(() => {
        navigate(`/restaurant/${restaurantId}/admin`);
      }, 1200);
    } catch (e) {
      console.error(e);
      setStatus("Failed to reset password. The token may be invalid or expired.");
    }
  };

  return (
    <div style={{ padding: 20 }}>
      <h2>Set a new password</h2>
      {!token ? <div>Invalid link</div> : (
        <div>
          <div><input type="password" placeholder="New password" value={password} onChange={e => setPassword(e.target.value)} /></div>
          <div><input type="password" placeholder="Confirm password" value={confirm} onChange={e => setConfirm(e.target.value)} /></div>
          <div style={{ marginTop: 12 }}><button onClick={submit}>Set password</button></div>
          <div style={{ marginTop: 12 }}>{status}</div>
        </div>
      )}
    </div>
  );
}
