<?php
    # Product name
    $config['product_name'] = 'BillionMail';

    $config['mime_types'] = '/var/roundcube/config/mime.types';
       
    # Plugins
    $config['plugins'] = array('password','userinfo','newmail_notifier','emoticons','zipdownload');

    # Password
    $config['password_query'] = 'update mailbox set password = %P where username = %u';
    $config['password_algorithm'] = 'md5-crypt';

    # Internal Postfix Submission via Docker Gateway
    $config['smtp_host'] = '172.66.1.1:2525';
    $config['smtp_user'] = '%u';
    $config['smtp_pass'] = '%p';



# Please create an extra file: extra.php. Used to persist overrides Configuration
