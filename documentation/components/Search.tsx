import { useState, useEffect } from 'react';
import Link from 'next/link';

interface Item {
  title: string;
  slug: string;
  text: string;
}

export default function Search() {
  const [query, setQuery] = useState('');
  const [items, setItems] = useState<Item[]>([]);
  const [results, setResults] = useState<Item[]>([]);

  useEffect(() => {
    fetch('/searchIndex.json')
      .then(res => res.json())
      .then(setItems)
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!query) {
      setResults([]);
      return;
    }
    const lower = query.toLowerCase();
    setResults(items.filter(i => i.title.toLowerCase().includes(lower) || i.text.toLowerCase().includes(lower)));
  }, [query, items]);

  return (
    <div className="search">
      <input
        type="text"
        placeholder="Search..."
        value={query}
        onChange={e => setQuery(e.target.value)}
      />
      {results.length > 0 && (
        <ul className="results">
          {results.map(item => (
            <li key={item.slug}>
              <Link href={item.slug}>{item.title}</Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
