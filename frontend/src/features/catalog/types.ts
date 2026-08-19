export type Book = {
  ID: string;
  Title: string;
  AuthorNames: string[] | null;
  CoverURL: string;
  Key: string;
  AuthorKeys: string[] | null;
  Description: string;
  Uploaded: boolean;
  Parsed: boolean;
  Prepared: boolean;
  ScriptGenerated: boolean;
};
