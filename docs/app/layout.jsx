import { Footer, Layout, Navbar } from 'nextra-theme-docs'
import { Banner, Head, Search } from 'nextra/components'
import { getPageMap } from 'nextra/page-map'
import 'nextra-theme-docs/style.css'

export const metadata = {
  title: {
    default: 'Goframe',
    template: '%s – Goframe'
  },
  description: 'GoFrame is a framework that is based on file generation.',
  openGraph: {
    title: 'Goframe',
    description: 'GoFrame is a framework that is based on file generation.',
    siteName: 'Goframe',
    locale: 'en_US',
    type: 'website'
  }
}

const navbar = (
  <Navbar
    logo={<span>Goframe</span>}
    projectLink="https://github.com/alexisvisco/goframe"
    chatLink="https://discord.com"
  />
)

const footer = (
  <Footer className="flex-col items-center md:items-start">
    Goframe Documentation
  </Footer>
)

export default async function RootLayout({ children }) {
  return (
    <html
      lang="en"
      dir="ltr"
      suppressHydrationWarning
    >
      <Head />
      <body>
        <Layout
          navbar={navbar}
          pageMap={await getPageMap()}
          docsRepositoryBase="https://github.com/alexisvisco/goframe/tree/main/docs"
          editLink="Edit this page on GitHub"
          sidebar={{ defaultMenuCollapseLevel: 1 }}
          footer={footer}
          search={<Search />}
        >
          {children}
        </Layout>
      </body>
    </html>
  )
}
