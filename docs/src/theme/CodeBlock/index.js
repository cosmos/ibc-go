import React from 'react';
import CodeBlock from '@theme-original/CodeBlock';

export default function CodeBlockWrapper(props) {
  const { source, ...codeBlockProps } = props;
  return (
    <>
      <CodeBlock {...codeBlockProps} />
      {source &&
        <div className='text-right mb-4'>
          <a href={source} target="_blank" rel="noopener noreferrer">View Source</a>
        </div>
      }
    </>
  );
}
