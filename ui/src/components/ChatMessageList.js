import React, { PropTypes } from 'react';
import ChatMessage from './ChatMessage';
import ImmutablePropTypes from 'react-immutable-proptypes';

const styles = {
    ul: {
        listStyle: 'none'
    }
};

const ChatMessageList = ({chatMessages}) => (
    <ul style={styles.ul}>
      {chatMessages.map((msg, idx)=>
       <ChatMessage {...msg}
          key={idx}
          displayFrom={!msg.previousSenderSame}
       />
      )}
    </ul>
);

ChatMessageList.propTypes = {
    chatMessages: ImmutablePropTypes.listOf(PropTypes.shape(
        ChatMessage.propTypes
    )).isRequired
};

export default ChatMessageList;
