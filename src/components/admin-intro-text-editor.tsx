"use client";

import { useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Link from "@tiptap/extension-link";
import { Bold, Italic, List, ListOrdered, Link as LinkIcon, Unlink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { getIntroText, saveIntroText } from "@/lib/api";

interface Props {
  rcNumber: string;
}

export function AdminIntroTextEditor({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<"ok" | "error" | null>(null);
  const [linkUrl, setLinkUrl] = useState("");
  const [linkOpen, setLinkOpen] = useState(false);
  const [loaded, setLoaded] = useState(false);

  const editor = useEditor({
    extensions: [
      StarterKit,
      Link.configure({ openOnClick: false, HTMLAttributes: { target: "_blank", rel: "noopener noreferrer" } }),
    ],
    content: "",
    editorProps: {
      attributes: {
        class: "min-h-[160px] px-3 py-2 text-sm focus:outline-none",
      },
    },
    onUpdate: () => setSaveResult(null),
  });

  useEffect(() => {
    if (!rcNumber || !session?.accessToken) return;
    setLoaded(false);
    getIntroText(rcNumber, session.accessToken)
      .then(({ introText }) => {
        editor?.commands.setContent(introText ?? "");
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [rcNumber, session?.accessToken, editor]);

  const applyLink = () => {
    if (!linkUrl) return;
    const url = linkUrl.startsWith("http") ? linkUrl : `https://${linkUrl}`;
    editor?.chain().focus().extendMarkRange("link").setLink({ href: url }).run();
    setLinkUrl("");
    setLinkOpen(false);
  };

  const handleSave = async () => {
    if (!editor) return;
    setSaving(true);
    setSaveResult(null);
    const html = editor.isEmpty ? null : editor.getHTML();
    try {
      await saveIntroText(rcNumber, html, session?.accessToken);
      setSaveResult("ok");
    } catch {
      setSaveResult("error");
    } finally {
      setSaving(false);
    }
  };

  if (!editor) return null;

  const ToolbarButton = ({
    onClick,
    active,
    title,
    children,
  }: {
    onClick: () => void;
    active?: boolean;
    title: string;
    children: React.ReactNode;
  }) => (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={[
        "p-1.5 rounded text-sm transition-colors",
        active
          ? "bg-primary text-primary-foreground"
          : "text-muted-foreground hover:bg-muted hover:text-foreground",
      ].join(" ")}
    >
      {children}
    </button>
  );

  return (
    <div className="space-y-3">
      <div className="border border-border rounded-md overflow-hidden">
        {/* Toolbar */}
        <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border bg-muted/40">
          <ToolbarButton
            title="Fett"
            active={editor.isActive("bold")}
            onClick={() => editor.chain().focus().toggleBold().run()}
          >
            <Bold className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            title="Kursiv"
            active={editor.isActive("italic")}
            onClick={() => editor.chain().focus().toggleItalic().run()}
          >
            <Italic className="h-4 w-4" />
          </ToolbarButton>
          <div className="w-px h-4 bg-border mx-1" />
          <ToolbarButton
            title="Aufzählung"
            active={editor.isActive("bulletList")}
            onClick={() => editor.chain().focus().toggleBulletList().run()}
          >
            <List className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            title="Nummerierte Liste"
            active={editor.isActive("orderedList")}
            onClick={() => editor.chain().focus().toggleOrderedList().run()}
          >
            <ListOrdered className="h-4 w-4" />
          </ToolbarButton>
          <div className="w-px h-4 bg-border mx-1" />
          <Popover open={linkOpen} onOpenChange={setLinkOpen}>
            <PopoverTrigger asChild>
              <button
                type="button"
                title="Link einfügen"
                className={[
                  "p-1.5 rounded text-sm transition-colors",
                  editor.isActive("link")
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground",
                ].join(" ")}
              >
                <LinkIcon className="h-4 w-4" />
              </button>
            </PopoverTrigger>
            <PopoverContent className="w-72 p-3" align="start">
              <p className="text-xs font-medium mb-2">Link-URL</p>
              <div className="flex gap-2">
                <Input
                  value={linkUrl}
                  onChange={(e) => setLinkUrl(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && applyLink()}
                  placeholder="https://..."
                  className="h-8 text-sm"
                />
                <Button size="sm" className="h-8" onClick={applyLink}>OK</Button>
              </div>
            </PopoverContent>
          </Popover>
          {editor.isActive("link") && (
            <ToolbarButton
              title="Link entfernen"
              onClick={() => editor.chain().focus().unsetLink().run()}
            >
              <Unlink className="h-4 w-4" />
            </ToolbarButton>
          )}
        </div>

        {/* Editor area */}
        <div className="prose prose-sm dark:prose-invert max-w-none [&_.ProseMirror]:focus:outline-none">
          <EditorContent editor={editor} />
        </div>
      </div>

      {!loaded && (
        <p className="text-xs text-muted-foreground">Lädt…</p>
      )}

      <div className="flex items-center gap-3">
        <Button onClick={handleSave} disabled={saving || !loaded} size="sm">
          {saving ? "Wird gespeichert…" : "Speichern"}
        </Button>
        {saveResult === "ok" && (
          <span className="text-sm text-green-600">Einleitungstext gespeichert</span>
        )}
        {saveResult === "error" && (
          <span className="text-sm text-destructive">Fehler beim Speichern</span>
        )}
      </div>
    </div>
  );
}
