import http from "k6/http";
import { check, sleep } from "k6";
import { SharedArray } from "k6/data";

const baseUrl = __ENV.BASE_URL || "http://localhost:8080/api/v1";
const examId = __ENV.EXAM_ID;
const peakVUs = Number(__ENV.PEAK_VUS || 2000);
const users = new SharedArray("students", () => JSON.parse(open(__ENV.STUDENTS_FILE || "./students.json")));

export const options = {
  scenarios: {
    exam_load: {
      executor: "per-vu-iterations",
      vus: peakVUs,
      iterations: 1,
      maxDuration: "30m",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<1000", "p(99)<2000"],
  },
};

export default function () {
  if (!examId || users.length < peakVUs) throw new Error("EXAM_ID and at least PEAK_VUS unique accounts in STUDENTS_FILE are required");
  const account = users[(__VU - 1) % users.length];
  const login = http.post(`${baseUrl}/auth/login`, JSON.stringify(account), { headers: { "Content-Type": "application/json" } });
  check(login, { "login succeeds": (response) => response.status === 200 });
  if (login.status !== 200) return;
  const token = login.json("data.access_token");
  const headers = { Authorization: `Bearer ${token}`, "Content-Type": "application/json" };
  const start = http.post(`${baseUrl}/student/exams/${examId}/start`, null, { headers: { ...headers, "Idempotency-Key": `start-${account.identifier}` } });
  check(start, { "attempt starts or resumes": (response) => response.status === 200 });
  if (start.status !== 200) return;
  const attempt = start.json("data");
  for (const question of attempt.questions.slice(0, 3)) {
    const answer = question.options[0];
    const save = http.put(`${baseUrl}/student/attempts/${attempt.attempt_id}/answers`, JSON.stringify({ exam_question_id: question.id, selected_option_id: answer.id }), { headers: { ...headers, "Idempotency-Key": `save-${attempt.attempt_id}-${question.id}` } });
    check(save, { "answer saves": (response) => response.status === 200 });
    sleep(1);
  }
  const submit = http.post(`${baseUrl}/student/attempts/${attempt.attempt_id}/submit`, null, { headers: { ...headers, "Idempotency-Key": `submit-${attempt.attempt_id}` } });
  check(submit, { "attempt submits": (response) => response.status === 200 });
}
