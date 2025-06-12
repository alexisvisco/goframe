import Link from 'next/link';
import Search from './Search';
import '../styles.css';

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="layout">
      <header className="header">
        <Link href="/">
          <h1>Goframe Docs</h1>
        </Link>
        <Search />
      </header>
      <main>{children}</main>
    </div>
  );
}
