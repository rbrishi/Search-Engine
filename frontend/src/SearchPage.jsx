import React, { useState } from "react";

function SearchPage() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState([]);
  const [count, setCount] = useState(0);
  const [time, setTime] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleSearch = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `http://localhost:8080/search?q=${encodeURIComponent(query)}`
      );
      if (!response.ok) {
        throw new Error("Search failed");
      }
      const data = await response.json();
      setResults(data.results);
      setCount(data.count);
      setTime(data.time_ms);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: "20px" }}>
      <h1>Search Engine</h1>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Enter search keywords"
        style={{ width: "300px", padding: "5px" }}
      />
      <button
        onClick={handleSearch}
        style={{ marginLeft: "10px", padding: "5px 10px" }}
      >
        Search
      </button>
      {loading && <p>Loading...</p>}
      {error && <p style={{ color: "red" }}>Error: {error}</p>}
      {results.length > 0 && (
        <div>
          <p>
            Found {count} results in {time}ms
          </p>
          <ul style={{ listStyleType: "none", padding: 0 }}>
            {results.map((result, index) => (
              <li key={index} style={{ margin: "10px 0" }}>
                <strong>Event ID:</strong> {result.EventId} <br />
                <strong>Message:</strong> {result.Message} <br />
                <strong>Timestamp:</strong> {result.NanoTimeStamp}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

export default SearchPage;
