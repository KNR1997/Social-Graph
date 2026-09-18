import PageContainer from '@/components/layout/page-container';
import { AiChatDemo } from '@/features/ai-chat/components/ai-chat-demo';
import { useDocumentTitle } from '@/hooks/use-document-title';

export default function AiChatPage() {
  useDocumentTitle('AI Chat');
  return (
    <PageContainer>
      <AiChatDemo />
    </PageContainer>
  );
}
