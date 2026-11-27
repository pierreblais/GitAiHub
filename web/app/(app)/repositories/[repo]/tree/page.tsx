import { TreeView } from "@/components/TreeView";

export default async function TreePage({ params, searchParams }: { params: Promise<{ repo: string }>; searchParams: Promise<{ ref?: string; path?: string }> }) {
  const { repo } = await params;
  const { ref, path } = await searchParams;
  return <TreeView repo={repo} gitRef={ref} path={path} />;
}
