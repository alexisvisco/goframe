const withMDX = require('@next/mdx')({
  extension: /\.md$/
});

module.exports = withMDX({
  pageExtensions: ['js', 'jsx', 'ts', 'tsx', 'md']
});
