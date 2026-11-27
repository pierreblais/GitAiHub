import { TreeView } from "@/components/TreeView";

export default async function RepoHome({ params }: { params: Promise<{ repo: string }> }) {
  const { repo } = await params;
  return <TreeView repo={repo} />;
}
