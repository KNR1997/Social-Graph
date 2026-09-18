import ChatViewPage from '@/features/chat/components/chat-view-page';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function ChatPage() {
  useDocumentTitle('Chat');
  return <ChatViewPage />;
}
