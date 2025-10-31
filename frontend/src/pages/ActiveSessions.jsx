import React from "react";

export default function ActiveSessions({ sessions = [], currentSessionId, onRevoke }) {
  if (!sessions || sessions.length === 0) {
    return <div>No active sessions</div>;
  }
  return (
    <div>
      {sessions.map(s => (
        <div key={s.id} style={{ borderBottom: "1px solid #eee", padding: 8, background: s.id === currentSessionId ? "#f6ffed" : "transparent" }}>
          <div>
            <strong>{s.adminEmail}</strong>
            {s.id === currentSessionId && <span style={{ marginLeft: 8, color: "#167a00", fontWeight: "600" }}>(This session)</span>}
          </div>
          <div>Created: {new Date(s.createdAt).toLocaleString()}</div>
          <div>Expires: {s.expiresAt ? new Date(s.expiresAt).toLocaleString() : "none"}</div>
          <div>IP: {s.ip || "—"} | UA: <small>{s.userAgent}</small></div>
          <div>Status: {s.revoked ? "revoked" : "active"}</div>
          <div style={{ marginTop: 6 }}>
            {!s.revoked && s.id !== currentSessionId ? <button onClick={() => onRevoke(s.id)}>Revoke</button> : null}
          </div>
        </div>
      ))}
    </div>
  );
}
