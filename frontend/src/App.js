import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import Admin from './pages/Admin';
import InviteAccept from './pages/InviteAccept';
import PasswordResetRequest from './pages/PasswordResetRequest';
import PasswordResetConfirm from './pages/PasswordResetConfirm';

export default function App() {
  const restaurantId = process.env.REACT_APP_DEFAULT_RESTAURANT || "1";
  
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home restaurantId={restaurantId} />} />
        <Route path="/restaurant/:id" element={<Home />} />
        <Route path="/restaurant/:id/admin" element={<Admin />} />
        <Route path="/invite/accept" element={<InviteAccept />} />
        <Route path="/password-reset/request" element={<PasswordResetRequest />} />
        <Route path="/password-reset/confirm" element={<PasswordResetConfirm />} />
      </Routes>
    </Router>
  );
}
