import { Badge } from "@multica/ui/components/ui/badge";
import { cn } from "@multica/ui/lib/utils";
import { ProviderLogo } from "./components/provider-logo";

const providerNameMap: Record<string, string> = {
  claude: "Claude Code",
  "claude-code": "Claude Code",
  codex: "Codex",
  opencode: "OpenCode",
  openclaw: "OpenClaw",
  hermes: "Hermes",
  droid: "Droid",
};

export function formatProviderName(provider: string): string {
  return providerNameMap[provider.toLowerCase()] ?? provider;
}

export function ProviderBadge({
  provider,
  className,
}: {
  provider: string;
  className?: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn("inline-flex items-center gap-1.5", className)}
    >
      <ProviderLogo provider={provider} className="h-3.5 w-3.5" />
      {formatProviderName(provider)}
    </Badge>
  );
}
