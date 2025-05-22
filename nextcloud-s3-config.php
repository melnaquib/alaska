<?php
$CONFIG = array (
  'objectstore' => array(
    'class' => '\\OC\\Files\\ObjectStore\\S3',
    'arguments' => array(
      'bucket' => 'nextcloud',
      'key'    => 'minioadmin',
      'secret' => 'minioadmin',
      'hostname' => 'localstack',
      'port' => 4566,
      'use_ssl' => false,
      'region' => 'us-east-1',
      'use_path_style' => true
    ),
  ),
); 