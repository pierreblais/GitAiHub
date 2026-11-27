import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

/** Renders Markdown safely: raw HTML in the source is not rendered. */
export function Markdown({ source, baseHref }: { source: string; baseHref?: string }) {
  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => {
            let target = href ?? "#";
            if (baseHref && target && !/^([a-z]+:|#|\/)/i.test(target)) target = baseHref + target;
            return <a href={target}>{children}</a>;
          },
        }}
      >
        {source}
      </ReactMarkdown>
    </div>
  );
}
