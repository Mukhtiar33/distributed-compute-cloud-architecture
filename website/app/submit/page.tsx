"use client";

import { useState } from "react";

export default function SubmitPage() {
  const [file, setFile] = useState<File | null>(null);
  const [jobType, setJobType] = useState<"single" | "parallel">("single");
  const [environment, setEnvironment] = useState("python-3.11");
  const [entrypoint, setEntrypoint] = useState("main.py");
  const [expectedOutput, setExpectedOutput] = useState("result.zip");
  const [inputs, setInputs] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<{ job_id: string; status: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file) {
      setError("Please select a zip file");
      return;
    }

    setSubmitting(true);
    setError(null);
    setResult(null);

    try {
      const manifest = {
        job_type: jobType,
        environment,
        entrypoint,
        ...(jobType === "parallel" && { inputs: inputs.split(",").map((s) => s.trim()).filter(Boolean) }),
        expected_output: expectedOutput,
      };

      const formData = new FormData();
      formData.append("manifest", JSON.stringify(manifest));
      formData.append("zip", file);

      const response = await fetch("/api/jobs", {
        method: "POST",
        body: formData,
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.validation_errors?.join(", ") || "Submission failed");
      }

      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Submission failed");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 text-white">
      <div className="mx-auto max-w-4xl px-6 py-24">
        <h1 className="text-4xl font-bold tracking-tight">Submit a Job</h1>
        <p className="mt-4 text-lg text-slate-300">
          Upload your project and let the platform handle the rest.
        </p>

        <form onSubmit={handleSubmit} className="mt-12 space-y-6">
          <div>
            <label className="block text-sm font-medium text-slate-300">Project Zip File</label>
            <input
              type="file"
              accept=".zip"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
              className="mt-2 block w-full rounded-lg bg-slate-800 px-4 py-3 text-white file:mr-4 file:rounded-lg file:border-0 file:bg-blue-600 file:px-4 file:py-2 file:text-white hover:file:bg-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300">Job Type</label>
            <select
              value={jobType}
              onChange={(e) => setJobType(e.target.value as "single" | "parallel")}
              className="mt-2 block w-full rounded-lg bg-slate-800 px-4 py-3 text-white"
            >
              <option value="single">Single Worker</option>
              <option value="parallel">Parallel (Multiple Workers)</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300">Environment</label>
            <select
              value={environment}
              onChange={(e) => setEnvironment(e.target.value)}
              className="mt-2 block w-full rounded-lg bg-slate-800 px-4 py-3 text-white"
            >
              <option value="python-3.11">Python 3.11</option>
              <option value="node-20">Node.js 20</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300">Entrypoint</label>
            <input
              type="text"
              value={entrypoint}
              onChange={(e) => setEntrypoint(e.target.value)}
              className="mt-2 block w-full rounded-lg bg-slate-800 px-4 py-3 text-white"
              placeholder="main.py"
            />
          </div>

          {jobType === "parallel" && (
            <div>
              <label className="block text-sm font-medium text-slate-300">Inputs (comma-separated)</label>
              <input
                type="text"
                value={inputs}
                onChange={(e) => setInputs(e.target.value)}
                className="mt-2 block w-full rounded-lg bg-slate-800 px-4 py-3 text-white"
                placeholder="input1.csv, input2.csv, input3.csv"
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-slate-300">Expected Output</label>
            <input
              type="text"
              value={expectedOutput}
              onChange={(e) => setExpectedOutput(e.target.value)}
              className="mt-2 block w-full rounded-lg bg-slate-800 px-4 py-3 text-white"
              placeholder="result.zip"
            />
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-blue-600 px-8 py-3 text-lg font-semibold hover:bg-blue-500 disabled:opacity-50 transition-colors"
          >
            {submitting ? "Submitting..." : "Submit Job"}
          </button>
        </form>

        {error && (
          <div className="mt-6 rounded-lg bg-red-900/50 p-4 text-red-200">
            <p className="font-semibold">Error</p>
            <p className="mt-1">{error}</p>
          </div>
        )}

        {result && (
          <div className="mt-6 rounded-lg bg-green-900/50 p-4 text-green-200">
            <p className="font-semibold">Job Submitted Successfully</p>
            <p className="mt-1">Job ID: {result.job_id}</p>
            <p className="mt-1">Status: {result.status}</p>
          </div>
        )}
      </div>
    </main>
  );
}
