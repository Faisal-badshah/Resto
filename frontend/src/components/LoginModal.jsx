import React, { useState } from "react";
import { login } from "../api";

export default function LoginModal({ restaurantId, onLogin }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");

  const submit = async () => {
    try {
      const res = await login(restaurantId, email, password);
      onLogin({ accessToken: res.token, role: res.role, currentSessionId: res.currentSessionId });
    } catch (e) {
      setErr("Login failed");
    }
  };

  return (
    <div style={{ padding: 12 }}>
      <h3>Admin Login</h3>
      <div><input placeholder="email" value={email} onChange={e => setEmail(e.target.value)} /></div>
      <div><input placeholder="password" type="password" value={password} onChange={e => setPassword(e.target.value)} /></div>
      <button onClick={submit}>Login</button>
      {err && <div style={{color:'red'}}>{err}</div>}
    </div>
  );
}
