import React, { PropTypes } from 'react';

const styles = {
    li: {
        whiteSpace: 'pre-wrap',
        display: 'block',
        paddingBottom: '10px',
        marginRight: '20px'
    },
    time: {
        color: '#BBB',
        fontSize: '0.8em',
        fontWeight: 200,
        marginLeft: '10px'
    },
    from: {
        margin: '20px 0px 0px 0px',
        display: 'block'
    },
    text: {
        fontSize: '0.9em'
    }
};

const ChatMessage = ({from, text, time, displayFrom}) => (
    <li style={styles.li}>
      {displayFrom ?
          <span style={styles.from}>
            <strong>{from}</strong> <span style={styles.time}>{time}</span>
          </span> :
          null
      }
      <div style={styles.text}>
        {text}
      </div>
    </li>
);

ChatMessage.propTypes = {
    from: PropTypes.string.isRequired,
    text: PropTypes.string.isRequired,
    time: PropTypes.string.isRequired,
    displayFrom: PropTypes.bool
};

export default ChatMessage;
