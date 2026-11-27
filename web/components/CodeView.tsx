import { highlightLines } from "@/lib/highlight";
import { languageFor } from "@/lib/lang";

export async function CodeView({ path, content }: { path: string; content: string }) {
  const text = content.endsWith("\n") ? content.slice(0, -1) : content;
  const lines = await highlightLines(text, languageFor(path));
  return (
    <div className="gh-code gh-highlight shiki py-2">
      <table>
        <tbody>
          {lines.map((html, i) => (
            <tr key={i} id={`L${i + 1}`}>
              <td className="gh-ln">
                <a href={`#L${i + 1}`}>{i + 1}</a>
              </td>
              <td dangerouslySetInnerHTML={{ __html: html || " " }} />
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
