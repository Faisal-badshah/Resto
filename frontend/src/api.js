import axios from "axios";

const API = axios.create({
  baseURL: process.env.REACT_APP_API_URL || "http://localhost:8080/api",
  withCredentials: true,
});

export function fetchRestaurant(id) {
  return API.get(`/restaurants/${id}`).then(r => r.data);
}

export function postOrder(restaurantId, payload) {
  return API.post(`/orders/${restaurantId}`, payload).then(r => r.data);
}

export function postSubscribe(restaurantId, email) {
  return API.post(`/subscribe/${restaurantId}`, { email }).then(r => r.data);
}

export function postReview(restaurantId, review) {
  return API.post(`/reviews/${restaurantId}`, review).then(r => r.data);
}

export function login(restaurantId, email, password) {
  return API.post("/login", { restaurantId, email, password }).then(r => r.data);
}

export function verify(token) {
  return API.get("/verify", { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}

export function adminGetOrders(restaurantId, token) {
  return API.get(`/admin/orders/${restaurantId}`, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}
export function adminUpdateMenus(restaurantId, payload, token) {
  return API.post(`/menus/${restaurantId}`, payload, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}
export function adminPatchRestaurant(restaurantId, payload, token) {
  return API.post(`/restaurants_patch/${restaurantId}`, payload, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}

export function adminInvite(restaurantId, email, role, token) {
  return API.post(`/admin/invite/${restaurantId}`, { email, role }, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}
export function acceptInvite(tokenStr, password) {
  return API.post(`/admin/invite/accept`, { token: tokenStr, password }).then(r => r.data);
}

export function requestPasswordReset(restaurantId, email) {
  return API.post(`/admin/password_reset/request`, { restaurantId, email }).then(r => r.data);
}
export function confirmPasswordReset(token, password) {
  return API.post(`/admin/password_reset/confirm`, { token, password }).then(r => r.data);
}

export function refreshSession() {
  return API.post("/refresh").then(r => r.data);
}
export function logout() {
  return API.post("/logout").then(r => r.data);
}

export function adminGetAudit(restaurantId, token) {
  return API.get(`/admin/audit/${restaurantId}`, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}

export function adminGetSessions(restaurantId, token) {
  return API.get(`/admin/sessions/${restaurantId}`, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}
export function adminRevokeSession(sessionId, token) {
  return API.post(`/admin/sessions/revoke`, { sessionId }, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}
export function adminRevokeOtherSessions(token) {
  return API.post(`/admin/sessions/revoke_all`, {}, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}

export function adminExportData(restaurantId, token) {
  return API.get(`/admin/export/${restaurantId}`, { headers: { Authorization: "Bearer " + token }, responseType: "blob" }).then(r => r.data);
}
export function adminExportMedia(restaurantId, token) {
  return API.get(`/admin/export_media/${restaurantId}`, { headers: { Authorization: "Bearer " + token }, responseType: "blob" }).then(r => r.data);
}
export function adminExportMediaToS3(restaurantId, payload, token) {
  return API.post(`/admin/export_media/${restaurantId}`, payload, { headers: { Authorization: "Bearer " + token } }).then(r => r.data);
}

export default API;
