export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Subjects often name a doc/feature and its scope on one line; raise the cap
    // from the conventional default of 100 to 150 so descriptive headers fit,
    // while still flagging an accidentally pasted blob.
    'header-max-length': [2, 'always', 150],
    // Bodies carry rationale and trade-off prose; raise the per-line cap so
    // explanatory bodies need not hard-wrap, while still catching a pasted blob.
    'body-max-line-length': [2, 'always', 1000],
  },
};