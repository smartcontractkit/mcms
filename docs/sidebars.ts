import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  tutorialSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Key Concepts',
      items: [
        'key-concepts/configuration',
        'key-concepts/mcm-proposal',
        'key-concepts/timelock-proposal',
        'key-concepts/chain-metadata',
        'key-concepts/operations',
        'key-concepts/merkle',
        'key-concepts/chain-family-sdk',
      ],
    },
    {
      type: 'category',
      label: 'Usage',
      items: [
        'usage/configuration',
        'usage/building-proposals',
        'usage/signing-proposals',
        'usage/executing-proposals',
        'usage/contract-inspection',
        'usage/timelock-proposal-flow',
      ],
    },
    {
      type: 'category',
      label: 'Contributing',
      items: [
        'contributing/integrating-new-chain-guide',
      ],
    },
  ],
};

export default sidebars;
